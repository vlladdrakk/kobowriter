package screener

import (
	"bytes"
	"image"
	"log"
	"math"

	"github.com/fogleman/gg"
	"github.com/olup/kobowriter/matrix"
	"github.com/shermp/go-fbink-v2/v2/gofbink"
)

type Screen struct {
	originalMatrix matrix.Matrix
	presentMatrix  matrix.Matrix
	fb             *gofbink.FBInk
	state          gofbink.FBInkState
	Width          int
	Height         int
	fontType       string
	ttSize         int
	otherWidth     int
	otherHeight    int
}

var dc = gg.NewContext(25, 40)
var charCache = map[string][]byte{}

func InitScreen(fontScale uint8) (s *Screen) {
	s = &Screen{}
	s.fontType = "bitmap"

	s.state = gofbink.FBInkState{}

	fbinkOpts := gofbink.FBInkConfig{}
	rOpts := gofbink.RestrictedConfig{
		Fontmult: fontScale,
		Fontname: gofbink.Ctrld,
	}
	s.fb = gofbink.New(&fbinkOpts, &rOpts)

	s.fb.Open()
	s.fb.Init(&fbinkOpts)

	// Setup fonts
	s.loadFonts(fontScale)

	s.fb.GetState(&fbinkOpts, &s.state)

	// clear screen on initialisation
	s.ClearFlash()

	s.Width = int(s.state.MaxCols)
	s.Height = int(s.state.MaxRows)

	// Set truetype height and width
	w, h := dc.MeasureString("A")
	s.otherWidth = int(s.state.ScreenWidth) / int(w)
	s.otherHeight = int(s.state.ScreenHeight) / int(h)

	s.presentMatrix = matrix.CreateNewMatrix(s.Width, s.Height)
	s.originalMatrix = matrix.CreateNewMatrix(s.Width, s.Height)

	println("Screen struct inited")

	return
}

func (s *Screen) loadFonts(fontScale uint8) {
	s.fb.AddOTfont("/mnt/onboard/SourceCodePro-Regular.ttf", gofbink.FntRegular)
	s.fb.AddOTfont("/mnt/onboard/SourceCodePro-Bold.ttf", gofbink.FntBold)

	s.ttSize = getTrueTypeSize(fontScale)

	font, err := gg.LoadFontFace("/mnt/onboard/SourceCodePro-Regular.ttf", float64(s.ttSize))

	if err != nil {
		log.Panicf("Font load error: %v", err)
	}

	dc.SetFontFace(font)
}

func (s *Screen) SetFontType(fontType string) {
	if s.fontType == fontType {
		return
	}

	w := s.Width
	h := s.Height

	s.Width = s.otherWidth
	s.Height = s.otherHeight

	s.otherWidth = w
	s.otherHeight = h

	s.fontType = fontType

	s.ClearFlash()
}

func getTrueTypeSize(fontScale uint8) int {
	switch fontScale {
	case 1:
		{
			return 20
		}
	case 2:
		{
			return 30
		}
	case 3:
		{
			return 40
		}
	}

	return 30
}

func (s *Screen) ChangeFontScale(scale uint8) {
	if scale > 3 {
		log.Println("Failed to change font scale, out of bounds: ", scale)
		return
	}

	_opts := gofbink.FBInkConfig{}
	opts := gofbink.RestrictedConfig{
		Fontmult: scale,
		Fontname: gofbink.Ctrld,
	}

	s.ttSize = getTrueTypeSize(scale)

	s.fb.UpdateRestricted(&_opts, &opts)
	// s.fb.ReInit(&_opts)
	s.RefreshFlash()
}

func (s *Screen) Clean() {
	s.fb.Close()
}

func (s *Screen) Print(matrix matrix.Matrix) {
	printDiff(s.presentMatrix, matrix, s.fb, s.fontType, s.ttSize)
	s.presentMatrix = matrix
}

func same(a matrix.MatrixElement, b matrix.MatrixElement) bool {
	return a.Content == b.Content && a.IsInverted == b.IsInverted
}

func printDiff(previous matrix.Matrix, next matrix.Matrix, fb *gofbink.FBInk, fontType string, ttSize int) {
	for i := range previous {
		for j := range previous[i] {
			if !same(previous[i][j], next[i][j]) {
				if fontType == "truetype" {
					ttWidth := ((ttSize / 5) * 3)
					fb.ClearScreen(&gofbink.FBInkConfig{
						IsInverted: next[i][j].IsInverted,
						NoRefresh:  true,
					}, &gofbink.FBInkRect{
						Top:    uint16(i * ttSize),
						Left:   uint16(j * ttWidth),
						Height: uint16(ttSize),
						Width:  uint16(ttWidth),
					})

					fontStyle := gofbink.FntRegular

					if next[i][j].Size > 0 {
						fontStyle = gofbink.FntBold
					}

					fb.PrintOT(string(next[i][j].Content), &gofbink.FBInkOTConfig{
						Margins: struct {
							Top    int16
							Bottom int16
							Left   int16
							Right  int16
						}{
							Top:  int16(i * ttSize),
							Left: int16(j * ttWidth),
						},
						SizePx:      uint16(ttSize + (2 * next[i][j].Size)),
						IsFormatted: false,
						Style:       fontStyle,
					}, &gofbink.FBInkConfig{IsInverted: next[i][j].IsInverted, NoRefresh: true})

				} else {
					fb.FBprint(string(next[i][j].Content), &gofbink.FBInkConfig{
						Row:        int16(i),
						Col:        int16(j),
						NoRefresh:  true,
						IsInverted: next[i][j].IsInverted,
					})
				}

			}

		}
	}

	fb.Refresh(0, 0, 0, 0, &gofbink.FBInkConfig{})
}

func (s *Screen) PrintPng(imgBytes []byte, w int, h int, x int, y int) {
	img, _, _ := image.Decode(bytes.NewReader(imgBytes))
	buffer, _ := getPixelsFromImage(img)
	s.fb.PrintRawData(buffer, w, h, uint16(x), uint16(y), &gofbink.FBInkConfig{})
}

func getCharImage(s string) []byte {
	if char, ok := charCache[s]; ok {
		return char
	} else {
		dc.SetRGB(1, 1, 1)
		dc.Clear()

		dc.SetRGB(0, 0, 0)
		dc.DrawString(s, 0, 35)
		img := dc.Image()
		buffer, _ := getPixelsFromImage(img)
		charCache[s] = buffer
		return buffer
	}
}

func (s *Screen) PrintAlert(message string, width int) {
	thisMatrix := matrix.CreateMatrixFromText(message, width)
	x := math.Floor((float64(s.state.MaxCols)/2)-float64(width)/2) - 1
	y := math.Floor((float64(s.state.MaxRows)/2)-float64(len(thisMatrix))/2) - 1
	outerMatrix := matrix.CreateNewMatrix(width+2, len(thisMatrix)+2)
	thisMatrix = matrix.PasteMatrix(outerMatrix, thisMatrix, 1, 1)
	thisMatrix = matrix.InverseMatrix(thisMatrix)
	s.Print(matrix.PasteMatrix(s.originalMatrix, thisMatrix, int(x), int(y)))
}

func (s *Screen) Clear() {
	s.fb.ClearScreen(&gofbink.FBInkConfig{}, &gofbink.FBInkRect{})
	s.presentMatrix = matrix.FillMatrix(s.presentMatrix, ' ')
}

func (s *Screen) ClearFlash() {
	s.fb.ClearScreen(&gofbink.FBInkConfig{IsFlashing: true}, &gofbink.FBInkRect{})
	s.presentMatrix = matrix.FillMatrix(s.presentMatrix, ' ')
}

func (s *Screen) RefreshFlash() {
	presenMatrix := s.presentMatrix
	s.ClearFlash()
	s.Print(presenMatrix)
}

func (s *Screen) GetOriginalMatrix() matrix.Matrix {
	return matrix.CopyMatrix(s.originalMatrix)
}

func (s *Screen) GetPresentMatrix() matrix.Matrix {
	return matrix.CopyMatrix(s.presentMatrix)
}
