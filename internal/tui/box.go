package tui

import (
	"fmt"
	"io"
	"strings"
)

const (
	BoxH  = "─"
	BoxV  = "│"
	BoxTL = "┌"
	BoxTR = "┐"
	BoxBL = "└"
	BoxBR = "┘"
)

func DrawBox(w io.Writer, row, col, height, width int) {
	MoveTo(w, row, col)
	fmt.Fprintf(w, "%s%s%s", BoxTL, strings.Repeat(BoxH, width-2), BoxTR)

	for r := row + 1; r < row+height-1; r++ {
		MoveTo(w, r, col)
		fmt.Fprintf(w, "%s%s%s", BoxV, strings.Repeat(" ", width-2), BoxV)
	}

	MoveTo(w, row+height-1, col)
	fmt.Fprintf(w, "%s%s%s", BoxBL, strings.Repeat(BoxH, width-2), BoxBR)
}

func DrawBoxTitle(w io.Writer, row, col, height, width int, title string) {
	DrawBox(w, row, col, height, width)
	titleStr := " " + title + " "
	titleCol := col + (width-len(titleStr))/2
	MoveTo(w, row, titleCol)
	fmt.Fprintf(w, "%s%s%s%s%s", Bold, FgWhite, titleStr, Reset, FgGray)
}
