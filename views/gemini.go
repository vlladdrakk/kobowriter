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
				url := strings.Clone(app.CurrentPage.Url)

				if strings.Contains(url, "?") {
					url = strings.Split(url, "?")[0]
				}

				re := regexp.MustCompile("[a-zA-Z](/)[a-zA-Z]")
				splitIndex := re.FindIndex([]byte(url))

				if len(splitIndex) > 0 {
					url = url[:splitIndex[0]+1]
				}

				app.LoadPage(url + link)
			} else {
				app.LoadPage(url + "/" + link)
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
			nextLink := app.FindNextLink()

			if nextLink >= 0 {
				text.SetCursorPos(Position{X: 0, Y: nextLink})
				// bus.Publish("GEMINI:render")
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
			log.Println("selected", selectedBookmark)
			stalledForInput = false
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
