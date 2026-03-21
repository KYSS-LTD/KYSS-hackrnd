package finalsentence

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	colorReset  = "\x1b[0m"
	colorWhite  = "\x1b[1;38;5;15m"
	colorGreen  = "\x1b[1;38;5;84m"
	colorRed    = "\x1b[1;38;5;203m"
	colorDim    = "\x1b[38;5;245m"
	colorBorder = "\x1b[38;5;110m"
	colorAccent = "\x1b[1;38;5;153m"
	colorGold   = "\x1b[1;38;5;221m"
)

type playerSnapshot struct {
	Nick       string
	LineIndex  int
	Typed      []typedRune
	Finished   bool
	FinishedAt time.Time
	Ratio      float64
}

type roundState struct {
	Lines       []string
	RoundNumber int
	Remaining   time.Duration
	Duration    time.Duration
	Players     []playerSnapshot
	LeaderRatio float64
	Winners     []string
}

func renderFrame(state roundState, viewerNick string) []byte {
	var buf bytes.Buffer
	viewer := state.player(viewerNick)

	buf.WriteString("\x1b[H")
	buf.WriteString(colorAccent)
	buf.WriteString("  FINAL SENTENCE")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	buf.WriteString(colorDim)
	buf.WriteString("  Печатай строки быстрее таймера и соперников. Новый игрок входит сразу, следующий раунд — на равных.\r\n")
	buf.WriteString(colorReset)

	buf.WriteString(renderMeter("TIMER", 58, 1-fraction(state.Remaining, state.Duration), true))
	buf.WriteString(renderMeter("PROGRESS", 58, safeRatio(viewer.Ratio), false))
	buf.WriteString("\r\n")
	buf.WriteString(renderTextPanel(state, viewer))
	buf.WriteString("\r\n")
	buf.WriteString(renderStandings(state, viewerNick))
	buf.WriteString("\r\n")
	buf.WriteString(colorDim)
	buf.WriteString("  Backspace — исправить символ. Q — выйти.\r\n")
	buf.WriteString("  ")
	buf.WriteString(winnerLabel(state.Winners))
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func (s roundState) player(nick string) playerSnapshot {
	for _, p := range s.Players {
		if p.Nick == nick {
			return p
		}
	}
	return playerSnapshot{}
}

func renderMeter(label string, width int, ratio float64, reverse bool) string {
	segments := len(roundTexts[0])
	fillWidth := width - (segments - 1)
	filled := int(math.Round(ratio * float64(fillWidth)))
	if filled < 0 {
		filled = 0
	}
	if filled > fillWidth {
		filled = fillWidth
	}

	var body strings.Builder
	remaining := filled
	for i := 0; i < segments; i++ {
		partWidth := fillWidth / segments
		if i < fillWidth%segments {
			partWidth++
		}
		partFill := min(remaining, partWidth)
		remaining -= partFill
		empty := partWidth - partFill
		if reverse {
			body.WriteString(colorRed)
			body.WriteString(strings.Repeat("█", partFill))
			body.WriteString(colorDim)
			body.WriteString(strings.Repeat("·", empty))
		} else {
			body.WriteString(colorGreen)
			body.WriteString(strings.Repeat("█", partFill))
			body.WriteString(colorDim)
			body.WriteString(strings.Repeat("·", empty))
		}
		if i < segments-1 {
			body.WriteString(colorBorder)
			body.WriteString("│")
		}
	}

	return fmt.Sprintf("  %s┌─ %s %s┐%s\r\n  %s│%s│%s\r\n  %s└%s┘%s\r\n",
		colorBorder, colorAccent, label, colorReset,
		colorBorder, body.String()+colorBorder, colorReset,
		colorBorder, strings.Repeat("─", width+2), colorReset,
	)
}

func renderTextPanel(state roundState, viewer playerSnapshot) string {
	var buf strings.Builder
	buf.WriteString(colorBorder)
	buf.WriteString("  ┌──────────────────────────────────────────────────────────────────────┐\r\n")
	buf.WriteString("  │")
	buf.WriteString(colorAccent)
	buf.WriteString(center(" ROUND TEXT ", 70))
	buf.WriteString(colorBorder)
	buf.WriteString("│\r\n")
	buf.WriteString("  ├──────────────────────────────────────────────────────────────────────┤\r\n")

	for i, line := range state.Lines {
		buf.WriteString("  │ ")
		switch {
		case i < viewer.LineIndex:
			buf.WriteString(colorGreen)
			buf.WriteString(padRight(line, 68))
		case i == viewer.LineIndex && !viewer.Finished:
			buf.WriteString(renderActiveLine(line, viewer.Typed, 68))
		case i == viewer.LineIndex && viewer.Finished:
			buf.WriteString(colorGreen)
			buf.WriteString(padRight(line, 68))
		default:
			buf.WriteString(colorWhite)
			buf.WriteString(padRight(line, 68))
		}
		buf.WriteString(colorBorder)
		buf.WriteString(" │\r\n")
	}
	buf.WriteString("  └──────────────────────────────────────────────────────────────────────┘")
	buf.WriteString(colorReset)
	return buf.String()
}

func renderActiveLine(line string, typed []typedRune, width int) string {
	target := []rune(line)
	var buf strings.Builder
	for i, r := range target {
		switch {
		case i < len(typed):
			if typed[i].correct {
				buf.WriteString(colorGreen)
			} else {
				buf.WriteString(colorRed)
			}
			buf.WriteRune(typed[i].value)
		case i == len(typed):
			buf.WriteString(colorAccent)
			buf.WriteRune(r)
		default:
			buf.WriteString(colorWhite)
			buf.WriteRune(r)
		}
	}
	visible := len(target)
	if visible < width {
		buf.WriteString(colorDim)
		buf.WriteString(strings.Repeat(" ", width-visible))
	}
	return buf.String()
}

func renderStandings(state roundState, viewerNick string) string {
	var buf strings.Builder
	buf.WriteString(colorBorder)
	buf.WriteString("  ┌──────────────────────────────────────────────────────────────────────┐\r\n")
	buf.WriteString("  │")
	buf.WriteString(colorAccent)
	buf.WriteString(center(" LOBBY PROGRESS ", 70))
	buf.WriteString(colorBorder)
	buf.WriteString("│\r\n")
	buf.WriteString("  ├──────────────────────────────────────────────────────────────────────┤\r\n")
	if len(state.Players) == 0 {
		buf.WriteString("  │ ")
		buf.WriteString(colorDim)
		buf.WriteString(padRight("waiting for players", 68))
		buf.WriteString(colorBorder)
		buf.WriteString(" │\r\n")
	} else {
		leader := state.leaderNick()
		for _, p := range state.Players {
			label := p.Nick
			if p.Nick == viewerNick {
				label += " (you)"
			}
			status := progressStatus(p)
			buf.WriteString("  │ ")
			buf.WriteString(colorWhite)
			buf.WriteString(padRight(label, 18))
			buf.WriteString(colorDim)
			buf.WriteString(" ")
			buf.WriteString(renderRaceBar(p.Ratio, state.LeaderRatio, p.Nick == leader))
			buf.WriteString(colorDim)
			buf.WriteString(" ")
			buf.WriteString(padRight(status, 9))
			buf.WriteString(colorBorder)
			buf.WriteString(" │\r\n")
		}
	}
	buf.WriteString("  └──────────────────────────────────────────────────────────────────────┘")
	buf.WriteString(colorReset)
	return buf.String()
}

func (s roundState) leaderNick() string {
	leader := ""
	best := -1.0
	for _, p := range s.Players {
		if p.Ratio > best {
			best = p.Ratio
			leader = p.Nick
		}
	}
	return leader
}

func renderRaceBar(playerRatio, leaderRatio float64, isLeader bool) string {
	const width = 28
	playerFill := int(math.Round(playerRatio * width))
	leaderFill := int(math.Round(leaderRatio * width))
	if playerFill < 0 {
		playerFill = 0
	}
	if leaderFill < playerFill {
		leaderFill = playerFill
	}
	if leaderFill > width {
		leaderFill = width
	}
	var buf strings.Builder
	buf.WriteString(colorBorder)
	buf.WriteString("[")
	for i := 0; i < width; i++ {
		switch {
		case i < playerFill:
			buf.WriteString(colorGreen)
			buf.WriteString("█")
		case i < leaderFill && !isLeader:
			buf.WriteString(colorAccent)
			buf.WriteString("·")
		default:
			buf.WriteString(colorDim)
			buf.WriteString("·")
		}
	}
	buf.WriteString(colorBorder)
	buf.WriteString("]")
	return buf.String()
}

func progressStatus(p playerSnapshot) string {
	if p.Finished {
		return "FINISHED"
	}
	return fmt.Sprintf("%3.0f%%", p.Ratio*100)
}

func fraction(remaining, total time.Duration) float64 {
	if total <= 0 {
		return 1
	}
	elapsed := total - remaining
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > total {
		elapsed = total
	}
	return float64(elapsed) / float64(total)
}

func safeRatio(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func center(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	left := (width - len(runes)) / 2
	right := width - len(runes) - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
