package calculator

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/asaskevich/EventBus"
	gcalc "github.com/mnogu/go-calculator"
	"github.com/olup/kobowriter/event"
	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/ui"
	"github.com/olup/kobowriter/utils"
)

func LaunchCalculator(screen *screener.Screen, bus EventBus.Bus) func() {
	stalledForInput := false
	text := &ui.TextView{
		Width:       int(screen.Width) - 4,
		Height:      int(screen.Height) - 2,
		CursorIndex: 0,
	}

	varMap := make(map[string]float64)

	onEvent := func(e event.KeyEvent) {
		if stalledForInput {
			return
		}

		if e.IsChar {
			text.SetContent(utils.InsertAt(text.Content, e.KeyChar, text.CursorIndex))
			text.SetCursorIndex(text.CursorIndex + 1)
		}
		// if is modifier key
		switch e.KeyValue {
		case "KEY_BACKSPACE":
			text.SetContent(utils.DeleteAt(text.Content, text.CursorIndex))
			text.SetCursorIndex(text.CursorIndex - 1)
		case "KEY_DEL":
			if text.CursorIndex < utils.LenString(text.Content) {
				text.SetContent(utils.DeleteAt(text.Content, text.CursorIndex+1))
			}
		case "KEY_SPACE":
			text.SetContent(utils.InsertAt(text.Content, " ", text.CursorIndex))
			text.SetCursorIndex(text.CursorIndex + 1)
		case "KEY_ENTER":
			currentLine := text.WrapContent[text.CursorPos.Y]
			text.SetCursorPos(utils.Position{X: len(currentLine), Y: text.CursorPos.Y})
			varRegex := regexp.MustCompile("[a-zA-Z][a-zA-Z0-9]* = .+")

			calculate := func(input string) float64 {
				for k, v := range varMap {
					if strings.Contains(input, k) {
						input = strings.ReplaceAll(input, k, fmt.Sprintf("%f", v))
					}
				}
				val, err := gcalc.Calculate(input)

				if err != nil {
					log.Printf("Failed calculation: %v\n", err)
				}

				return val
			}

			if varRegex.Match([]byte(currentLine)) {
				parts := strings.Split(currentLine, "=")
				name := strings.Trim(parts[0], " ")
				value := strings.Trim(parts[1], " ")

				calculatedValue := calculate(value)

				varMap[name] = calculatedValue
				text.SetContent(utils.InsertAt(text.Content, "\n", text.CursorIndex))
				text.SetCursorIndex(text.CursorIndex + 1)
				text.SetContent(utils.InsertAt(text.Content, fmt.Sprintf("# %s = %f\n", name, varMap[name]), text.CursorIndex))
				text.SetCursorPos(utils.Position{X: 0, Y: text.CursorPos.Y + 1})

			} else {
				val := calculate(currentLine)
				text.SetContent(utils.InsertAt(text.Content, fmt.Sprintf(" = %f\n", val), text.CursorIndex))
				text.SetCursorPos(utils.Position{X: 0, Y: text.CursorPos.Y + 1})
			}
		case "KEY_RIGHT":
			text.SetCursorIndex(text.CursorIndex + 1)
		case "KEY_LEFT":
			text.SetCursorIndex(text.CursorIndex - 1)
		case "KEY_DOWN":
			text.SetCursorPos(utils.Position{
				X: text.CursorPos.X,
				Y: text.CursorPos.Y,
			})
		case "KEY_UP":
			text.SetCursorPos(utils.Position{
				X: text.CursorPos.X,
				Y: text.CursorPos.Y,
			})
		case "KEY_ESC":
			bus.Publish("ROUTING", "menu")
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
