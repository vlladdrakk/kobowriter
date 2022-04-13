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
	Content     string         `json:"content"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	WrapContent []string       `json:"wrap_content"`
	CursorIndex int            `json:"cursor_index"`
	CursorPos   Position       `json:"cursor_pos"`
	LineCount   []int          `json:"lineCount"`
	Scroll      int            `json:"scroll"`
	LinkMap     map[int]string `json:"linkMap"`
}

func (t *HyperTextView) Init(width int, height int) {
	t.Width = width
	t.Height = height
	t.Content = ""
	t.Scroll = 0
	t.LinkMap = make(map[int]string)
}

func (t *HyperTextView) SetContent(text string) {
	t.parseLinks(text)

	lineCount := []int{}
	for _, line := range t.WrapContent {
		lineCount = append(lineCount, utf8.RuneCountInString(line)+1)
	}
	t.LineCount = lineCount
}

func (t *HyperTextView) GetWrappedContent() []string {
	return t.WrapContent
}

func (t *HyperTextView) GetCurrentLine() string {
	return t.WrapContent[t.CursorPos.Y]
}

func (t *HyperTextView) SetCursorIndex(index int) {

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
	t.updateScroll()

}

func (t *HyperTextView) RenderMatrix() matrix.Matrix {
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

func (t *HyperTextView) updateScroll() {
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

	t.WrapContent = strings.Split(utils.WrapText(parsedBody, t.Width), "\n")
	t.Content = parsedBody
	c := 0

	for i, line := range t.WrapContent {
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
