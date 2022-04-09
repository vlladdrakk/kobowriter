package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asaskevich/EventBus"
	gbrowser "github.com/olup/kobowriter/apps"
	"github.com/olup/kobowriter/event"
	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/ui"
)

func LaunchGemini(screen *screener.Screen, bus EventBus.Bus, url string) func() {
	app := gbrowser.GeminiBrowser{
		CurrentPage: gbrowser.Page{
			Url:     strings.Clone(url),
			LinkMap: make(map[int]string),
		},
		Cache:       make(map[string]gbrowser.Page),
		Bus:         bus,
		ScreenWidth: int(screen.Width) - 2,
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
			app.PushHistory(app.CurrentPage)

			bus.Publish("GEMINI:load", link, 0)
			app.LoadPage(link)
		}

		if !strings.Contains(link, "://") {
			app.PushHistory(app.CurrentPage)

			app.LoadPage(app.CurrentPage.Url + "/" + link)
		}
	}

	bus.SubscribeAsync("GEMINI:handleLink", linkHandler, false)

	updateCursor := func() {
		text.setCursorPos(app.CurrentPage.Position)
	}

	bus.SubscribeAsync("GEMINI:update_cursor", updateCursor, false)

	render := func() {
		text.setContent(app.CurrentPage.Body)
		text.setCursorIndex(0)

		text.setCursorPos(app.CurrentPage.Position)
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
				X: text.cursorPos.X,
				Y: text.cursorPos.Y + linesToMove,
			})
		case "KEY_UP":
			text.setCursorPos(Position{
				X: text.cursorPos.X,
				Y: text.cursorPos.Y - linesToMove,
			})
		case "KEY_ESC":
			bus.Publish("ROUTING", "menu")
		case "KEY_ENTER":
			linkMap := app.CurrentPage.LinkMap
			lineNumber := text.cursorPos.Y
			fmt.Println("line number:", lineNumber, linkMap[lineNumber])
			if _, ok := linkMap[lineNumber]; ok {
				bus.Publish("GEMINI:handleLink", linkMap[lineNumber])
			}
		case "KEY_F12":
			screen.ClearFlash()
		case "KEY_TAB":
			linkMap := app.CurrentPage.LinkMap
			lineNumber := text.cursorPos.Y
			keys := make([]int, 0, len(linkMap))
			for k := range linkMap {
				keys = append(keys, k)
			}

			sort.Ints(keys)
			for _, k := range keys {
				if k > lineNumber && linkMap[k] != linkMap[lineNumber] {
					text.setCursorPos(Position{
						X: text.cursorPos.X,
						Y: k,
					})
					break
				}
			}
		case "g":
			stalledForInput = true
			goToUrl := ui.PromptForInput(screen, bus, "Go to url:")
			stalledForInput = false
			fmt.Println("goto", goToUrl)
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
			fmt.Println("bookmark count", len(bookmarks))
			selectedBookmark := ui.MultiSelect(screen, bus, "Bookmarks", bookmarks)
			fmt.Println("selected", selectedBookmark)
			stalledForInput = false
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
