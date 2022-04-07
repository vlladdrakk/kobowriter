package views

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

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

type Page struct {
	url      string
	body     string
	linkMap  map[int]string
	exp      time.Time
	position Position
}

type bookmark struct {
	url  string
	name string
}

type GemState struct {
	cache       map[string]Page
	currentPage Page
	history     []HistoryItem
	future      []HistoryItem
	bookmarks   []bookmark
	bus         EventBus.Bus
	screenWidth int
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

func (s *GemState) PushHistory(p Page) {
	item := HistoryItem{
		url:      p.url,
		position: p.position,
	}

	s.history = append(s.history, item)
}

func (s *GemState) PushFuture(p Page) {
	item := HistoryItem{
		url:      p.url,
		position: p.position,
	}

	s.future = append(s.future, item)
}

func (s *GemState) PopHistory() HistoryItem {
	// Pop the last url off the stack
	if len(s.history) > 0 {
		var item HistoryItem
		item, s.history = s.history[len(s.history)-1], s.history[:len(s.history)-1]

		return item
	} else {
		return HistoryItem{}
	}
}

func (s *GemState) PopFuture() HistoryItem {
	// Pop the last url off the stack
	if len(s.future) > 0 {
		var item HistoryItem
		item, s.future = s.future[len(s.future)-1], s.future[:len(s.future)-1]

		return item
	} else {
		return HistoryItem{}
	}
}

// Checks cache and loads the currentPage, renders to the screen
func (s *GemState) LoadPage(url string) {
	var p Page
	cachedPage, isCached := s.cache[url]

	if isCached && cachedPage.exp.After(time.Now()) {
		p = cachedPage
	} else {
		p = s.LoadUrl(url)
		// Cache the page
		s.cache[url] = p
	}

	s.currentPage = p

	s.bus.Publish("GEMINI:render")
}

// No cache check, just loads a URL and returns a Page struct
func (s *GemState) LoadUrl(url string) Page {
	response := makeRequest(url)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v", err)
	}

	content, linkMap := parseGemText(string(body), s.screenWidth)
	return Page{
		body:    content,
		url:     url,
		exp:     time.Now().Add(5 * time.Minute),
		linkMap: linkMap,
	}
}

func (s *GemState) GoBack() {
	s.PushFuture(s.currentPage)
	item := s.PopHistory()

	s.LoadPage(item.url)
	s.SetCursor(item.position)
}

func (s *GemState) GoForward() {
	s.PushHistory(s.currentPage)
	item := s.PopFuture()

	s.LoadPage(item.url)
	s.SetCursor(item.position)
}

func (s *GemState) SetCursor(p Position) {
	s.currentPage.position = p

	s.bus.Publish("GEMINI:update_cursor")
}

func LaunchGemini(screen *screener.Screen, bus EventBus.Bus, url string) func() {
	state := GemState{
		currentPage: Page{
			url:     strings.Clone(url),
			linkMap: make(map[int]string),
		},
		cache:       make(map[string]Page),
		bus:         bus,
		screenWidth: int(screen.Width) - 2,
	}

	text := &TextView{
		width:       int(screen.Width) - 4,
		height:      int(screen.Height) - 2,
		content:     "",
		scroll:      0,
		cursorIndex: 0,
	}

	linkHandler := func(link string) {
		if strings.Contains(link, "http://") || strings.Contains(link, "https://") {
			return
		}

		if strings.Contains(link, "gemini://") {
			state.PushHistory(state.currentPage)

			bus.Publish("GEMINI:load", link, 0)
			state.LoadPage(link)
		}

		if !strings.Contains(link, "://") {
			state.PushHistory(state.currentPage)

			state.LoadPage(state.currentPage.url + "/" + link)
		}
	}

	bus.SubscribeAsync("GEMINI:handleLink", linkHandler, false)

	updateCursor := func() {
		text.setCursorPos(state.currentPage.position)
	}

	bus.SubscribeAsync("GEMINI:update_cursor", updateCursor, false)

	render := func() {
		text.setContent(state.currentPage.body)
		text.setCursorIndex(0)

		text.setCursorPos(state.currentPage.position)
		compiledMatrix := matrix.PasteMatrix(screen.GetOriginalMatrix(), text.renderMatrix(), 2, 1)
		screen.Print(compiledMatrix)
	}

	bus.SubscribeAsync("GEMINI:render", render, false)

	bus.Publish("GEMINI:handleLink", url)
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
			linkMap := state.currentPage.linkMap
			lineNumber := text.cursorPos.y
			fmt.Println("line number:", lineNumber, linkMap[lineNumber])
			if _, ok := linkMap[lineNumber]; ok {
				bus.Publish("GEMINI:handleLink", linkMap[lineNumber])
			}
		case "KEY_F12":
			screen.ClearFlash()
		case "KEY_TAB":
			linkMap := state.currentPage.linkMap
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

			bus.Publish("GEMINI:handleLink", goToUrl)
		case "u":
			state.GoBack()
		case "f":
			state.GoForward()
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
