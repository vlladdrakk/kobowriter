package gbrowser

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/a-h/gemini"
	"github.com/asaskevich/EventBus"
	"github.com/olup/kobowriter/utils"
)

type Position = utils.Position

type HistoryItem struct {
	url      string
	position Position
}

type Page struct {
	Url      string
	Body     string
	LinkMap  map[int]string
	exp      time.Time
	Position Position
}

type bookmark struct {
	url  string
	name string
}

type GeminiBrowser struct {
	Cache       map[string]Page
	CurrentPage Page
	History     []HistoryItem
	Future      []HistoryItem
	Bookmarks   []bookmark
	Bus         EventBus.Bus
	ScreenWidth int
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

func (s *GeminiBrowser) PushHistory(p Page) {
	item := HistoryItem{
		url:      p.Url,
		position: p.Position,
	}

	s.History = append(s.History, item)
}

func (s *GeminiBrowser) PushFuture(p Page) {
	item := HistoryItem{
		url:      p.Url,
		position: p.Position,
	}

	s.Future = append(s.Future, item)
}

func (s *GeminiBrowser) PopHistory() HistoryItem {
	// Pop the last url off the stack
	if len(s.History) > 0 {
		var item HistoryItem
		item, s.History = s.History[len(s.History)-1], s.History[:len(s.History)-1]

		return item
	} else {
		return HistoryItem{}
	}
}

func (s *GeminiBrowser) PopFuture() HistoryItem {
	// Pop the last url off the stack
	if len(s.Future) > 0 {
		var item HistoryItem
		item, s.Future = s.Future[len(s.Future)-1], s.Future[:len(s.Future)-1]

		return item
	} else {
		return HistoryItem{}
	}
}

// Checks cache and loads the currentPage, renders to the screen
func (s *GeminiBrowser) LoadPage(url string) {
	var p Page
	cachedPage, isCached := s.Cache[url]

	if isCached && cachedPage.exp.After(time.Now()) {
		p = cachedPage
	} else {
		p = s.LoadUrl(url)
		// Cache the page
		s.Cache[url] = p
	}

	s.CurrentPage = p

	s.Bus.Publish("GEMINI:render")
}

// No cache check, just loads a URL and returns a Page struct
func (s *GeminiBrowser) LoadUrl(url string) Page {
	response := makeRequest(url)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v", err)
	}

	content, linkMap := parseGemText(string(body), s.ScreenWidth)
	return Page{
		Body:    content,
		Url:     url,
		exp:     time.Now().Add(5 * time.Minute),
		LinkMap: linkMap,
	}
}

func (s *GeminiBrowser) GoBack() {
	s.PushFuture(s.CurrentPage)
	item := s.PopHistory()

	s.LoadPage(item.url)
	s.SetCursor(item.position)
}

func (s *GeminiBrowser) GoForward() {
	s.PushHistory(s.CurrentPage)
	item := s.PopFuture()

	s.LoadPage(item.url)
	s.SetCursor(item.position)
}

func (s *GeminiBrowser) SetCursor(p Position) {
	s.CurrentPage.Position = p

	s.Bus.Publish("GEMINI:update_cursor")
}

func parseGemText(body string, width int) (string, map[int]string) {
	linkMap := make(map[int]string)
	var parsedBody string
	lineNum := 0

	for _, line := range strings.Split(body, "\n") {
		if len(line) < 3 || line[0:2] != "=>" {
			wrappedLines := utils.WrapLine(line, width)
			parsedBody = parsedBody + wrappedLines + "\n"
			lineNum += len(strings.Split(wrappedLines, "\n")) + 1
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
