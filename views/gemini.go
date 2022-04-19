package views

import (
	"log"
	"regexp"
	"strings"

	"github.com/asaskevich/EventBus"
	gbrowser "github.com/olup/kobowriter/apps"
	"github.com/olup/kobowriter/event"
	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/ui"
)

func LaunchGemini(screen *screener.Screen, bus EventBus.Bus, url string, saveLocation string) func() {
	app, ok := gbrowser.LoadState(saveLocation)
	stalledForInput := false

	screen.SetFontType("truetype")

	if !ok {
		app = gbrowser.GeminiBrowser{
			CurrentPage: gbrowser.Page{
				Url: strings.Clone(url),
			},
			Cache:        make(map[string]gbrowser.Page),
			Bus:          bus,
			ScreenWidth:  int(screen.Width) - 4,
			ScreenHeight: int(screen.Height) - 2,
			SaveLocation: saveLocation,
		}
	} else {
		app.Bus = bus
		app.Cache = make(map[string]gbrowser.Page)
		app.ScreenWidth = int(screen.Width) - 4
		app.ScreenHeight = int(screen.Height) - 2
		app.SaveLocation = saveLocation
	}

	text := &ui.HyperTextView{
		Width:       int(screen.Width) - 4,
		Height:      int(screen.Height) - 2,
		CursorIndex: 0,
	}

	linkHandler := func(link string) {
		if strings.Contains(link, "http://") || strings.Contains(link, "https://") {
			return
		}

		if strings.Contains(link, "gemini://") {
			app.PushHistory(app.CurrentPage)

			bus.Publish("GEMINI:load", link, 0)
			app.LoadPage(link)
		}

		if !strings.Contains(link, "://") {
			app.PushHistory(app.CurrentPage)

			if link[0] == '/' {
				baseUrl := strings.Clone(app.CurrentPage.Url)

				if strings.Contains(baseUrl, "?") {
					baseUrl = strings.Split(baseUrl, "?")[0]
				}

				re := regexp.MustCompile("[a-zA-Z](/)[a-zA-Z]")
				splitIndex := re.FindIndex([]byte(baseUrl))

				if len(splitIndex) > 0 {
					baseUrl = baseUrl[:splitIndex[0]+1]
				}

				app.LoadPage(baseUrl + link)
			} else {
				app.LoadPage(app.CurrentPage.Url + "/" + link)
			}
		}
	}

	bus.SubscribeAsync("GEMINI:handleLink", linkHandler, false)

	inputHandler := func(prompt string, url string) {
		stalledForInput = true
		input := ui.PromptForInput(screen, bus, prompt)
		bus.Publish("GEMINI:handleLink", url+"?"+input)
		stalledForInput = false
	}

	bus.SubscribeAsync("GEMINI:input", inputHandler, false)

	updateCursor := func() {
		text.SetCursorPos(app.CurrentPage.Position)
	}

	bus.SubscribeAsync("GEMINI:update_cursor", updateCursor, false)

	render := func() {
		text = &app.CurrentPage.View

		compiledMatrix := matrix.PasteMatrix(screen.GetOriginalMatrix(), text.RenderMatrix(), 2, 1)
		screen.Print(compiledMatrix)
	}

	bus.SubscribeAsync("GEMINI:render", render, false)

	// bus.Publish("GEMINI:handleLink", url)
	bus.Publish("GEMINI:render")

	onEvent := func(e event.KeyEvent) {
		if stalledForInput {
			return
		}
		linesToMove := 1
		if e.IsCtrl {
			linesToMove = text.Height
		}

		// if is modifier key
		switch e.KeyValue {
		case "KEY_RIGHT":
			text.SetCursorIndex(text.CursorIndex + 1)
		case "KEY_LEFT":
			text.SetCursorIndex(text.CursorIndex - 1)
		case "KEY_DOWN":
			text.SetCursorPos(Position{
				X: text.CursorPos.X,
				Y: text.CursorPos.Y + linesToMove,
			})
		case "KEY_UP":
			text.SetCursorPos(Position{
				X: text.CursorPos.X,
				Y: text.CursorPos.Y - linesToMove,
			})
		case "KEY_ESC":
			app.SaveState()
			screen.SetFontType("bitmap")
			bus.Publish("ROUTING", "menu")
		case "KEY_ENTER":
			linkMap := app.CurrentPage.View.LinkMap
			lineNumber := text.CursorPos.Y
			log.Println("line number:", lineNumber, linkMap[lineNumber])
			if _, ok := linkMap[lineNumber]; ok {
				bus.Publish("GEMINI:handleLink", linkMap[lineNumber])
			}
		case "KEY_F12":
			screen.ClearFlash()
		case "KEY_TAB":
			var nextLink int

			if e.IsShift {
				nextLink = app.FindPrevLink()
			} else {
				nextLink = app.FindNextLink()
			}

			if nextLink >= 0 {
				text.SetCursorPos(Position{X: 0, Y: nextLink})
			}
		case "g":
			stalledForInput = true
			goToUrl := ui.PromptForInput(screen, bus, "Go to url:")
			stalledForInput = false

			if goToUrl == "" {
				break
			}

			if !strings.Contains(goToUrl, "gemini://") {
				goToUrl = "gemini://" + goToUrl
			}

			bus.Publish("GEMINI:handleLink", goToUrl)
		case "u":
			app.GoBack()
		case "f":
			app.GoForward()
		case "a":
			stalledForInput = true
			bookmarkName := ui.PromptForInput(screen, bus, "Name for the bookmark")
			stalledForInput = false
			app.BookmarkCurrent(bookmarkName)
		case "b":
			stalledForInput = true
			bookmarks := app.GetBookmarkOptions()
			selectedBookmark := ui.MultiSelect(screen, bus, "Bookmarks", bookmarks)
			bus.Publish("GEMINI:handleLink", selectedBookmark)
			stalledForInput = false
		case "r":
			bus.Publish("GEMINI:handleLink", app.CurrentPage.Url)
		case "R":
			delete(app.Cache, app.CurrentPage.Url)
			bus.Publish("GEMINI:handleLink", app.CurrentPage.Url)
		}

		compiledMatrix := matrix.PasteMatrix(screen.GetOriginalMatrix(), text.RenderMatrix(), 2, 1)
		screen.Print(compiledMatrix)
	}

	bus.SubscribeAsync("KEY", onEvent, false)

	// display
	bus.Publish("KEY", event.KeyEvent{})

	return func() {
		bus.Unsubscribe("KEY", onEvent)
	}
}
