package pingpong

import (
	"bytes"
	"fmt"
	"strings"
)

type frameState struct {
	nick        string
	side        side
	leftPaddle  int
	rightPaddle int
	ballX       int
	ballY       int
	leftScore   int
	rightScore  int
	status      string
	players     []string
	leftName    string
	rightName   string
}

func renderFrame(state frameState) []byte {
	var buf bytes.Buffer
	buf.WriteString("\x1b[H")
	buf.WriteString("  TERMINAL PING PONG\r\n")
	buf.WriteString(fmt.Sprintf("  %s [%d]  :  [%d] %s\r\n", strings.ToUpper(state.leftName), state.leftScore, state.rightScore, strings.ToUpper(state.rightName)))
	buf.WriteString(fmt.Sprintf("  роль: %s\r\n", roleLabel(state.side)))
	buf.WriteString("  ┌")
	buf.WriteString(strings.Repeat("─", boardWidth))
	buf.WriteString("┐\r\n")

	for y := 1; y <= boardHeight; y++ {
		buf.WriteString("  │")
		for x := 1; x <= boardWidth; x++ {
			buf.WriteString(cellAt(state, x, y))
		}
		buf.WriteString("│\r\n")
	}

	buf.WriteString("  └")
	buf.WriteString(strings.Repeat("─", boardWidth))
	buf.WriteString("┘\r\n")
	buf.WriteString("  W/S или ↑/↓ — двигать ракетку. Q — выход. R — рестарт после матча.\r\n")
	buf.WriteString(fmt.Sprintf("  статус: %s\r\n", state.status))
	buf.WriteString(fmt.Sprintf("  в лобби: %s\r\n", strings.Join(state.players, ", ")))
	return buf.Bytes()
}

func cellAt(state frameState, x, y int) string {
	if x == boardWidth/2 {
		if y%2 == 0 {
			return "┊"
		}
		return " "
	}
	if x == 2 && y >= state.leftPaddle && y < state.leftPaddle+paddleHeight {
		return "█"
	}
	if x == boardWidth-1 && y >= state.rightPaddle && y < state.rightPaddle+paddleHeight {
		return "█"
	}
	if x == state.ballX && y == state.ballY {
		return "●"
	}
	return " "
}

func roleLabel(side side) string {
	switch side {
	case sideLeft:
		return "левая ракетка"
	case sideRight:
		return "правая ракетка"
	default:
		return "зритель"
	}
}
