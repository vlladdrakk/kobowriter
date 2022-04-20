package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/utils"
)

type TextView struct {
	Content     string
	Width       int
	Height      int
	WrapContent []string
	CursorIndex int
	CursorPos   Position
	LineCount   []int
	Scroll      int
}

func (t *TextView) init(width int) {
	t.Width = width
}

func (t *TextView) SetContent(text string) {
	t.Content = text
	t.WrapContent = strings.Split(utils.WrapText(text, t.Width), "\n")

	lineCount := []int{}
	for _, line := range t.WrapContent {
		lineCount = append(lineCount, utf8.RuneCountInString(line)+1)
	}
	t.LineCount = lineCount
}

func (t *TextView) GetCurrentLine() string {
	return t.WrapContent[t.CursorPos.Y]
}

func (t *TextView) SetCursorIndex(index int) {

	// Bounds
	if index < 0 {
		index = 0
	}
	if index > utils.LenString(t.Content) {
		index = utils.LenString(t.Content)
	}

	// Processing
	t.CursorIndex = index
	x := 0
	y := 0

	agg := 0

	for i, count := range t.LineCount {
		aggNext := count + agg
		if aggNext > t.CursorIndex {
			y = i
			x = t.CursorIndex - agg
			break
		}
		agg = aggNext
	}

	t.CursorPos = Position{
		X: x,
		Y: y,
	}

	t.UpdateScroll()

}

func (t *TextView) SetCursorPos(position Position) {
	// Bounds
	if position.Y < 0 {
		position.Y = 0
	}

	if position.X < 0 {
		position.X = 0
	}

	if position.Y > len(t.LineCount)-1 {
		position.Y = len(t.LineCount) - 1
	}

	if t.LineCount[position.Y]-1 < position.X {
		position.X = t.LineCount[position.Y] - 1
	}

	// Procesing

	agg := 0

	for i := 0; i < position.Y; i++ {
		agg += t.LineCount[i]
	}

	agg += position.X

	t.CursorPos = position
	t.CursorIndex = agg
	t.UpdateScroll()

}

func (t *TextView) RenderMatrix() matrix.Matrix {
	textMatrix := matrix.CreateMatrixFromText(t.Content, t.Width)
	if t.CursorPos.X >= 0 && t.CursorPos.Y >= 0 && t.CursorPos.X < t.Width {
		textMatrix[t.CursorPos.Y][t.CursorPos.X].IsInverted = true
	}
	endBound := t.Scroll + t.Height
	if endBound > len(textMatrix) {
		endBound = len(textMatrix)
	}
	scrolledTextMatrix := textMatrix[t.Scroll:endBound]
	return scrolledTextMatrix
}

func (t *TextView) UpdateScroll() {
	y := t.CursorPos.Y

	if y > t.Scroll+t.Height-1 {
		t.Scroll = y - 5
	}
	if y < t.Scroll {
		t.Scroll = y - t.Height + 5
	}
	if t.Scroll > len(t.WrapContent) {
		t.Scroll = len(t.WrapContent) - 5
	}
	if t.Scroll < 0 {
		t.Scroll = 0
	}
}
