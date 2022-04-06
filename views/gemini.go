package views

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/a-h/gemini"
	"github.com/asaskevich/EventBus"
	"github.com/olup/kobowriter/event"
	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/utils"
)

type HistoryItem struct {
	url      string
	position Position
}

func parseDomain(url string) string {
	domain := strings.Replace(url, "gemini://", "", 1)

	if strings.Contains(domain, "/") {
		domain = strings.Split(domain, "/")[0]
	}

	return domain
}

func makeRequest(url string) *gemini.Response {
	client := gemini.NewClient()
	ctx := context.Background()

	if !strings.Contains(url, "gemini://") {
		url = "gemini://" + url
	}

	// Make initial request
	// TODO: handle authentication
	r, certificates, _, ok, err := client.Request(ctx, url)
	for !ok && err == nil {
		// If the client is missing the server certs
		if len(certificates) > 0 {
			for i := range certificates {
				client.AddServerCertificate(parseDomain(url), certificates[i])
			}
		}

		// Try the request again
		r, certificates, _, ok, err = client.Request(ctx, url)
	}

	if err != nil {
		fmt.Println("Request failed:", err)
	}

	// Follow redirects
	if r != nil && r.Header.Code[0] == '3' {
		fmt.Println("Redirecting to", r.Header.Meta)
		return makeRequest(r.Header.Meta)
	}

	return r
}

func LaunchGemini(screen *screener.Screen, bus EventBus.Bus, url string) func() {
	var linkMap map[int]string
	var currentUrl string = url
	var history []HistoryItem
	text := &TextView{
		width:       int(screen.Width) - 4,
		height:      int(screen.Height) - 2,
		content:     "",
		scroll:      0,
		cursorIndex: 0,
	}

	loadHandler := func(url string, pos int) {
		response := makeRequest(url)
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("failed to read body: %v", err)
		}

		currentUrl = url

		var content string
		content, linkMap = parseGemText(string(body), text.width)
		text.setContent(content)
		text.setCursorIndex(0)

		text.setCursorPos(Position{x: 0, y: pos})
		compiledMatrix := matrix.PasteMatrix(screen.GetOriginalMatrix(), text.renderMatrix(), 2, 1)
		screen.Print(compiledMatrix)
	}

	linkHandler := func(link string) {
		if strings.Contains(link, "http://") || strings.Contains(link, "https://") {
			return
		}

		if strings.Contains(link, "gemini://") {
			history = append(history, HistoryItem{url: currentUrl, position: text.cursorPos})
			bus.Publish("GEMINI:load", link, 0)
		}

		if !strings.Contains(link, "://") {
			history = append(history, HistoryItem{url: currentUrl, position: text.cursorPos})
			bus.Publish("GEMINI:load", currentUrl+"/"+link, 0)
		}
	}

	bus.SubscribeAsync("GEMINI:load", loadHandler, false)
	bus.SubscribeAsync("GEMINI:handleLink", linkHandler, false)
	bus.Publish("GEMINI:load", url, 0)
	stalledForInput := false

	onEvent := func(e event.KeyEvent) {
		if stalledForInput {
			return
		}
		linesToMove := 1
		if e.IsCtrl {
			linesToMove = text.height
		}

		// if is modifier key
		switch e.KeyValue {
		// case "KEY_ENTER":
		// text.setContent(utils.InsertAt(text.content, "\n", text.cursorIndex))
		// text.setCursorIndex(text.cursorIndex + 1)
		case "KEY_RIGHT":
			text.setCursorIndex(text.cursorIndex + 1)
		case "KEY_LEFT":
			text.setCursorIndex(text.cursorIndex - 1)
		case "KEY_DOWN":
			text.setCursorPos(Position{
				x: text.cursorPos.x,
				y: text.cursorPos.y + linesToMove,
			})
		case "KEY_UP":
			text.setCursorPos(Position{
				x: text.cursorPos.x,
				y: text.cursorPos.y - linesToMove,
			})
		case "KEY_ESC":
			bus.Publish("ROUTING", "menu")
		case "KEY_ENTER":
			lineNumber := text.cursorPos.y
			fmt.Println("line number:", lineNumber, linkMap[lineNumber])
			if _, ok := linkMap[lineNumber]; ok {
				bus.Publish("GEMINI:handleLink", linkMap[lineNumber])
			}
		case "KEY_F12":
			screen.ClearFlash()
		case "KEY_TAB":
			lineNumber := text.cursorPos.y
			keys := make([]int, 0, len(linkMap))
			for k := range linkMap {
				keys = append(keys, k)
			}

			sort.Ints(keys)
			for _, k := range keys {
				if k > lineNumber && linkMap[k] != linkMap[lineNumber] {
					text.setCursorPos(Position{
						x: text.cursorPos.x,
						y: k,
					})
					break
				}
			}
		case "g":
			stalledForInput = true
			goToUrl := getInput(screen, bus, "Go to url:")
			stalledForInput = false

			bus.Publish("GEMINI:load", goToUrl, 0)
		case "u":
			// Pop the last url off the stack
			var lastUrl HistoryItem
			lastUrl, history = history[len(history)-1], history[:len(history)-1]
			bus.Publish("GEMINI:load", lastUrl.url, lastUrl.position.y)
		}

		compiledMatrix := matrix.PasteMatrix(screen.GetOriginalMatrix(), text.renderMatrix(), 2, 1)
		screen.Print(compiledMatrix)
	}

	bus.SubscribeAsync("KEY", onEvent, false)

	// display
	bus.Publish("KEY", event.KeyEvent{})

	return func() {
		bus.Unsubscribe("KEY", onEvent)
	}
}

func getInput(s *screener.Screen, bus EventBus.Bus, prompt string) string {
	writePrompt := func(input string) {
		// Set base layer
		m := s.GetOriginalMatrix()
		// Add the current input to the matrix
		if input != "" {
			m = matrix.PasteMatrix(
				m,
				matrix.CreateMatrixFromText(input, utils.LenString(input)),
				15,
				4,
			)
		}
		// Add the prompt message
		topMatrix := matrix.CreateMatrixFromText(prompt+"\n"+strings.Repeat("=", utils.LenString(prompt)), utils.LenString(prompt))
		// merge the base and top matrices
		m = matrix.PasteMatrix(m, topMatrix, 4, 4)

		s.Print(m)
	}

	var result string
	c := make(chan bool)
	defer close(c)

	writePrompt("")

	onKey := func(e event.KeyEvent) {
		if e.IsChar {
			result = result + e.KeyChar
		} else {
			switch e.KeyValue {
			case "KEY_ENTER":
				fmt.Println("Done inputting")
				c <- true
			case "KEY_BACKSPACE":
				result = result[:len(result)-1]
			}
		}

		writePrompt(result)
	}

	bus.SubscribeAsync("KEY", onKey, false)

	// display
	bus.Publish("KEY", event.KeyEvent{})

	for done := range c {
		if done {
			break
		}
	}

	bus.Unsubscribe("KEY", onKey)

	return result
}

func parseGemText(body string, width int) (string, map[int]string) {
	linkMap := make(map[int]string)
	var parsedBody string
	lineNum := 0

	for _, line := range strings.Split(body, "\n") {
		if len(line) < 3 || line[0:2] != "=>" {
			parsedBody = parsedBody + line + "\n"
			lineNum++
			continue
		}

		parts := strings.Fields(line)
		linkText := strings.Join(parts[2:], " ")
		newLine := utils.WrapLine("=> "+linkText+"\n", width)

		for _, l := range strings.Split(newLine, "\n") {
			parsedBody = parsedBody + l + "\n"
			linkMap[lineNum] = strings.Clone(parts[1])
			lineNum++
		}
	}

	return parsedBody, linkMap
}
