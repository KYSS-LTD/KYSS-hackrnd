package tui

import (
	"fmt"
	"io"
	"strings"
)

const (
	Reset  = "\x1b[0m"
	Bold   = "\x1b[1m"
	Dim    = "\x1b[2m"

	FgWhite  = "\x1b[97m"
	FgGray   = "\x1b[90m"
	FgBlack  = "\x1b[30m"
	BgBlack  = "\x1b[40m"
	BgGray   = "\x1b[100m"

	AltScreenOn  = "\x1b[?1049h"
	AltScreenOff = "\x1b[?1049l"
	CursorHide   = "\x1b[?25l"
	CursorShow   = "\x1b[?25h"
	ClearScreen  = "\x1b[2J"
	Home         = "\x1b[H"
)

func MoveTo(w io.Writer, row, col int) {
	fmt.Fprintf(w, "\x1b[%d;%dH", row, col)
}

func CenterPad(s string, width int) string {
	l := len(s)
	if l >= width {
		return s
	}
	left := (width - l) / 2
	right := width - l - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func Repeat(s string, n int) string {
	return strings.Repeat(s, n)
}
