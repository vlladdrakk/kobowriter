package ui

import (
	"fmt"
	"strings"

	"github.com/asaskevich/EventBus"
	"github.com/olup/kobowriter/event"
	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/screener"
	"github.com/olup/kobowriter/utils"
)

type SelectOption struct {
	Label string
	Value string
}

var promptWidth int = 40

func PromptForInput(s *screener.Screen, bus EventBus.Bus, prompt string) string {
	cursorPos := 0
	writePrompt := func(input string, prevWindowStart int) int {
		inputWindow := strings.Clone(input)
		isScrollable := len(input) > (promptWidth - 4)
		windowStart := 0

		if isScrollable {
			if prevWindowStart <= cursorPos && cursorPos <= (prevWindowStart+promptWidth-4) {
				windowStart = prevWindowStart
			} else {
				// The cursor is at the end of the input, the window is at the end of the input
				if len(input) == cursorPos {
					windowStart = len(input) - (promptWidth - 4)
				}

				// If the cursor is not within the last len(window) then shift the window to include the cursor
				if cursorPos < len(input)-(promptWidth-4) {
					windowStart = cursorPos
				}
			}

			inputWindow = input[windowStart:(windowStart + (promptWidth - 4))]
		}

		whitespacePaddingLen := (promptWidth - 3) - len(inputWindow)

		if whitespacePaddingLen < 0 {
			whitespacePaddingLen = 0
		}
		// Set base layer
		m := s.GetOriginalMatrix()
		// Add the current input to the matrix
		m = matrix.PasteMatrix(
			m,
			matrix.CreateMatrixFromText(strings.Repeat("*", promptWidth)+"\n* "+inputWindow+strings.Repeat(" ", whitespacePaddingLen)+"*\n"+strings.Repeat("*", promptWidth), s.Width),
			2,
			5,
		)

		// Find the position of the cursor inside the input window
		relativeCursor := cursorPos - windowStart

		// Add cursor
		m = matrix.PasteMatrix(
			m,
			matrix.InverseMatrix(
				matrix.CreateMatrixFromText(" ", 1),
			),
			4+relativeCursor,
			6,
		)

		// Add the prompt message
		topMatrix := matrix.CreateMatrixFromText(prompt, s.Width)
		// merge the base and top matrices
		m = matrix.PasteMatrix(m, topMatrix, 2, 4)

		s.Print(m)

		return windowStart
	}

	var result string
	c := make(chan bool)
	defer close(c)

	windowStart := 0
	writePrompt("", windowStart)

	onKey := func(e event.KeyEvent) {
		if e.IsChar {
			if len(result) == cursorPos {
				result = result + e.KeyChar
			} else {
				result = result[:cursorPos] + e.KeyChar + result[cursorPos:]
			}

			if cursorPos <= len(result) {
				cursorPos++
			}
		} else {
			switch e.KeyValue {
			case "KEY_ENTER":
				fmt.Println("Done inputting")
				c <- true
			case "KEY_BACKSPACE":
				if len(result) == cursorPos {
					result = result[:len(result)-1]
				} else {
					result = result[:cursorPos-1] + result[cursorPos:]
				}
				if cursorPos > 0 {
					cursorPos = (cursorPos - 1) % len(result)
				}
			case "KEY_ESC":
				c <- true
			case "KEY_LEFT":
				if cursorPos > 0 {
					cursorPos = (cursorPos - 1) % len(result)
				}
			case "KEY_RIGHT":
				if cursorPos <= len(result) {
					cursorPos++
				}
			case "KEY_DOWN":
				cursorPos = len(result)
			case "KEY_UP":
				cursorPos = 0
			}
		}

		windowStart = writePrompt(result, windowStart)
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

func MultiSelect(screen *screener.Screen, bus EventBus.Bus, prompt string, options []SelectOption) string {
	selected := 0

	var result string
	c := make(chan bool)
	defer close(c)

	onKey := func(e event.KeyEvent) {

		if e.KeyValue == "KEY_UP" && selected > 0 {
			selected--
		}
		if e.KeyValue == "KEY_DOWN" && selected < len(options)-1 {
			selected++
		}

		if e.KeyValue == "KEY_ENTER" {
			result = options[selected].Value
			c <- true
		}

		line := 1

		matrixx := screen.GetOriginalMatrix()
		matrixx = matrix.PasteMatrix(matrixx, matrix.CreateMatrixFromText(prompt+"\n"+strings.Repeat("=", utils.LenString(prompt)), utils.LenString(prompt)), 4, line)

		line += 2

		for i, option := range options {
			optionMatrix := matrix.CreateMatrixFromText(option.Label, utils.LenString(option.Label))
			if selected == i {
				optionMatrix = matrix.InverseMatrix(optionMatrix)
			}
			matrixx = matrix.PasteMatrix(matrixx, optionMatrix, 4, line+i)
		}

		screen.Print(matrixx)
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
