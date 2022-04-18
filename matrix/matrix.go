package matrix

import (
	"strings"
	"unicode"

	"github.com/olup/kobowriter/utils"
)

type MatrixElement struct {
	Content    rune
	IsInverted bool
	Size       int
}

type Matrix [][]MatrixElement

func CreateNewMatrix(width int, height int) Matrix {
	a := make([][]MatrixElement, height)
	for i := range a {
		a[i] = make([]MatrixElement, width)
		for j := range a[i] {
			a[i][j] = MatrixElement{
				Content:    ' ',
				IsInverted: false,
				Size:       0,
			}
		}
	}
	return a
}

func CreateMatrixFromText(text string, width int) Matrix {

	wrapped := utils.WrapText(text, int(width))

	wrapedArray := strings.Split(wrapped, "\n")
	result := CreateNewMatrix(width, len(wrapedArray))

	for i := range result {
		for j := range result[i] {
			if j < utils.LenString(wrapedArray[i]) {
				result[i][j].Content = []rune(wrapedArray[i])[j]
			}
		}
	}

	return result
}

func PasteMatrix(baseMatrix Matrix, topMatrix Matrix, offsetX int, offsetY int) Matrix {
	resultMatrix := CopyMatrix(baseMatrix)
	for i := range resultMatrix {
		localI := i - offsetY
		if localI < 0 || localI >= len(topMatrix) {
			continue
		}

		for j := range resultMatrix[i] {
			localJ := j - offsetX
			if localJ < 0 || localJ >= len(topMatrix[localI]) {
				continue
			}

			resultMatrix[i][j] = topMatrix[localI][localJ]

		}
	}

	return resultMatrix
}

func MatrixToText(matrix Matrix) string {
	stringz := make([]string, len(matrix))
	for i := range matrix {
		for _, elem := range matrix[i] {
			stringz[i] = stringz[i] + string(elem.Content)
		}
	}
	return strings.Join(stringz, "")
}

func InverseMatrix(in Matrix) (out Matrix) {
	out = in
	for i := range out {
		for j, elem := range out[i] {
			out[i][j].IsInverted = !elem.IsInverted
		}
	}
	return
}

func FillMatrix(in Matrix, char rune) (out Matrix) {
	out = CopyMatrix(in)
	for i := range out {
		for j := range out[i] {
			out[i][j].Content = char
		}
	}
	return
}

func CopyMatrix(in Matrix) (out Matrix) {
	if len(in) == 0 {
		return Matrix{}
	}
	out = CreateNewMatrix(len(in[0]), len(in))
	for i := range out {
		for j := range out[i] {
			out[i][j].Content = in[i][j].Content
			out[i][j].IsInverted = in[i][j].IsInverted
			out[i][j].Size = in[i][j].Size
		}
	}
	return
}

func SetLineSize(m Matrix, line int, size int) Matrix {
	if len(m) < line {
		return m
	}

	for i := range m[line] {
		m[line][i].Size = size
		m[line][i].Content = unicode.ToUpper(m[line][i].Content)
	}

	return m
}
