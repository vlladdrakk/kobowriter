package gbrowser

import (
	"context"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/a-h/gemini"
	"github.com/asaskevich/EventBus"
	"github.com/olup/kobowriter/ui"
	"github.com/olup/kobowriter/utils"
)

type Position = utils.Position
type HyperTextView = ui.HyperTextView

type HistoryItem struct {
	Url      string `json:"url"`
	Position Position
}

type Page struct {
	Url      string        `json:"url"`
	View     HyperTextView `json:"view"`
	Exp      time.Time     `json:"expiration"`
	Position Position      `json:"position"`
}

type Bookmark struct {
	Url  string `json:"url"`
	Name string `json:"name"`
}

type GeminiBrowser struct {
	Cache        map[string]Page `json:"cache"`
	CurrentPage  Page            `json:"current_page"`
	History      []HistoryItem   `json:"history"`
	Future       []HistoryItem   `json:"future"`
	Bookmarks    []Bookmark      `json:"bookmarks"`
	Bus          EventBus.Bus
	ScreenWidth  int
	ScreenHeight int
	SaveLocation string `json:"save_location"`
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
		log.Println("Request failed:", err)
	}

	// Follow redirects
	if r != nil && r.Header.Code[0] == '3' {
		log.Println("Redirecting to", r.Header.Meta)
		return makeRequest(r.Header.Meta)
	}

	return r
}

func (s *GeminiBrowser) PushHistory(p Page) {
	item := HistoryItem{
		Url:      p.Url,
		Position: p.Position,
	}

	s.History = append(s.History, item)
}

func (s *GeminiBrowser) PushFuture(p Page) {
	item := HistoryItem{
		Url:      p.Url,
		Position: p.Position,
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

	if isCached && cachedPage.Exp.After(time.Now()) {
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
	log.Println("Loaded", url)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("failed to read body: %v", err)
	}

	view := HyperTextView{}
	view.Init(s.ScreenWidth, s.ScreenHeight)
	view.SetContent(string(body))

	return Page{
		View: view,
		Url:  url,
		Exp:  time.Now().Add(5 * time.Minute),
	}
}

func (s *GeminiBrowser) GoBack() {
	s.PushFuture(s.CurrentPage)
	item := s.PopHistory()

	s.LoadPage(item.Url)
	s.SetCursor(item.Position)
}

func (s *GeminiBrowser) GoForward() {
	s.PushHistory(s.CurrentPage)
	item := s.PopFuture()

	s.LoadPage(item.Url)
	s.SetCursor(item.Position)
}

func (s *GeminiBrowser) SetCursor(p Position) {
	s.CurrentPage.Position = p

	s.Bus.Publish("GEMINI:update_cursor")
}

func (s *GeminiBrowser) BookmarkCurrent(name string) {
	s.Bookmarks = append(s.Bookmarks, Bookmark{
		Url:  s.CurrentPage.Url,
		Name: name,
	})
}

func (s *GeminiBrowser) GetBookmarkOptions() []ui.SelectOption {
	var bookmarkOptions []ui.SelectOption
	for _, b := range s.Bookmarks {
		bookmarkOptions = append(bookmarkOptions, ui.SelectOption{
			Label: b.Name,
			Value: b.Url,
		})
	}

	return bookmarkOptions
}

func (s *GeminiBrowser) FindNextLink() int {
	return s.CurrentPage.View.FindNextLink()
}
