package ui

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/olup/kobowriter/matrix"
	"github.com/olup/kobowriter/utils"
)

type Position = utils.Position

type HyperTextView struct {
	content     string
	Width       int
	Height      int
	wrapContent []string
	CursorIndex int
	CursorPos   Position
	lineCount   []int
	scroll      int
	LinkMap     map[int]string
}

func (t *HyperTextView) Init(width int, height int) {
	t.Width = width
	t.Height = height
	t.content = ""
	t.scroll = 0
	t.LinkMap = make(map[int]string)
}

func (t *HyperTextView) SetContent(text string) {
	t.parseLinks(text)

	lineCount := []int{}
	for _, line := range t.wrapContent {
		lineCount = append(lineCount, utf8.RuneCountInString(line)+1)
	}
	t.lineCount = lineCount
}

func (t *HyperTextView) GetWrappedContent() []string {
	return t.wrapContent
}

func (t *HyperTextView) GetCurrentLine() string {
	return t.wrapContent[t.CursorPos.Y]
}

func (t *HyperTextView) SetCursorIndex(index int) {

	// Bounds
	if index < 0 {
		index = 0
	}
	if index > utils.LenString(t.content) {
		index = utils.LenString(t.content)
	}

	// Processing
	t.CursorIndex = index
	x := 0
	y := 0

	agg := 0

	for i, count := range t.lineCount {
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

	t.updateScroll()

}

func (t *HyperTextView) SetCursorPos(position Position) {
	// Bounds
	if position.Y < 0 {
		position.Y = 0
	}

	if position.X < 0 {
		position.X = 0
	}

	if position.Y > len(t.lineCount)-1 {
		position.Y = len(t.lineCount) - 1
	}

	if t.lineCount[position.Y]-1 < position.X {
		position.X = t.lineCount[position.Y] - 1
	}

	// Procesing

	agg := 0

	for i := 0; i < position.Y; i++ {
		agg += t.lineCount[i]
	}

	agg += position.X

	t.CursorPos = position
	t.CursorIndex = agg
	t.updateScroll()

}

func (t *HyperTextView) RenderMatrix() matrix.Matrix {
	textMatrix := matrix.CreateMatrixFromText(t.content, t.Width)
	if t.CursorPos.X >= 0 && t.CursorPos.Y >= 0 && t.CursorPos.X < t.Width {
		textMatrix[t.CursorPos.Y][t.CursorPos.X].IsInverted = true
	}
	endBound := t.scroll + t.Height
	if endBound > len(textMatrix) {
		endBound = len(textMatrix)
	}
	scrolledTextMatrix := textMatrix[t.scroll:endBound]
	return scrolledTextMatrix
}

func (t *HyperTextView) updateScroll() {
	y := t.CursorPos.Y

	if y > t.scroll+t.Height-1 {
		t.scroll = y - 5
	}
	if y < t.scroll {
		t.scroll = y - t.Height + 5
	}
	if t.scroll > len(t.wrapContent) {
		t.scroll = len(t.wrapContent) - 5
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t *HyperTextView) parseLinks(content string) {
	var linkList []string
	var parsedBody string

	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}

		if len(line) < 3 || line[0:2] != "=>" {
			wrappedLines := utils.WrapLine(line, t.Width)
			parsedBody = parsedBody + wrappedLines + "\n"
			continue
		}

		parts := strings.Fields(line)
		linkText := strings.Join(parts[2:], " ")
		newLine := utils.WrapLine("=> "+linkText+"\n", t.Width)
		linkList = append(linkList, strings.Clone(parts[1]))

		for _, l := range strings.Split(newLine, "\n") {
			parsedBody = parsedBody + l + "\n"
		}
	}

	t.wrapContent = strings.Split(utils.WrapText(parsedBody, t.Width), "\n")
	t.content = parsedBody
	c := 0

	for i, line := range t.wrapContent {
		if len(line) >= 3 && line[0:2] == "=>" {
			t.LinkMap[i] = linkList[c]
			c++
		}
	}
}

// Finds the next link and returns the line number.
// Returns -1 if there is no link
func (t *HyperTextView) FindNextLink() int {
	linkLines := make([]int, 0, len(t.LinkMap))
	for k := range t.LinkMap {
		linkLines = append(linkLines, k)
	}

	sort.Ints(linkLines)
	for _, lineNum := range linkLines {
		if lineNum > t.CursorPos.Y {
			return lineNum
		}
	}

	return -1
}
