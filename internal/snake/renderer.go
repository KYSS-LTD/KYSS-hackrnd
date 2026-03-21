package snake

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

const (
	cellWidth  = 2
	panelWidth = 28

	cellEmpty = "  "
	cellBody  = "██"
	cellHead  = "▓▓"
	cellApple = "()"

	colorReset     = "\x1b[0m"
	colorBorder    = "\x1b[38;5;28m"
	colorInk       = "\x1b[38;5;22m"
	colorTitle     = "\x1b[1;38;5;22m"
	colorMuted     = "\x1b[38;5;28m"
	colorApple     = "\x1b[38;5;52m"
	colorPanelBg   = "\x1b[48;5;120m"
	colorArenaBg   = "\x1b[48;5;119m"
	colorOverlay   = "\x1b[1;38;5;22m"
	colorOverlayBg = "\x1b[48;5;150m"
	colorAlive     = "\x1b[1;38;5;22m"
	colorDead      = "\x1b[1;38;5;88m"
)

var snakePalette = []string{
	"\x1b[38;5;22m",
	"\x1b[38;5;28m",
	"\x1b[38;5;58m",
	"\x1b[38;5;94m",
	"\x1b[38;5;64m",
}

func renderFrame(state *GameState, dead map[string]bool, order []string, overlayNick string) []byte {
	var buf bytes.Buffer
	fieldWidth := Width * cellWidth
	overlay := deathOverlayLines(overlayNick)
	overlayWidth := 0
	if len(overlay) > 0 {
		overlayWidth = len([]rune(overlay[0]))
	}
	overlayRow, overlayCol := overlayOrigin(fieldWidth, len(overlay), overlayWidth)
	panelLines := renderPanelLines(state, dead, order, overlayNick)

	buf.WriteString("\x1b[H")
	buf.WriteString(colorTitle)
	buf.WriteString("  TERMINAL SNAKE")
	buf.WriteString(colorMuted)
	buf.WriteString(strings.Repeat(" ", max(2, fieldWidth-14)))
	buf.WriteString(colorTitle)
	buf.WriteString("OLD-SCHOOL ARENA")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	buf.WriteString(colorMuted)
	buf.WriteString("  monochrome UI inspired by terminal-snake")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")

	buf.WriteString(colorBorder)
	buf.WriteString("  ╔")
	buf.WriteString(strings.Repeat("═", fieldWidth))
	buf.WriteString("╗  ╔")
	buf.WriteString(strings.Repeat("═", panelWidth))
	buf.WriteString("╗")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")

	for y := 0; y < Height; y++ {
		buf.WriteString(colorBorder)
		buf.WriteString("  ║")
		buf.WriteString(colorReset)

		overlayLine := ""
		if y >= overlayRow && y < overlayRow+len(overlay) {
			overlayLine = overlay[y-overlayRow]
		}
		buf.WriteString(renderArenaRow(state, dead, y, overlayLine, overlayCol))

		buf.WriteString(colorBorder)
		buf.WriteString("║  ║")
		buf.WriteString(colorReset)
		buf.WriteString(renderPanelLine(panelLines, y))
		buf.WriteString(colorBorder)
		buf.WriteString("║")
		buf.WriteString(colorReset)
		buf.WriteString("\r\n")
	}

	buf.WriteString(colorBorder)
	buf.WriteString("  ╚")
	buf.WriteString(strings.Repeat("═", fieldWidth))
	buf.WriteString("╝  ╚")
	buf.WriteString(strings.Repeat("═", panelWidth))
	buf.WriteString("╝")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")

	buf.WriteString(colorMuted)
	buf.WriteString("  ←↑↓→ / WASD")
	buf.WriteString(colorInk)
	buf.WriteString(" move  ")
	buf.WriteString(colorMuted)
	buf.WriteString("C")
	buf.WriteString(colorInk)
	buf.WriteString(" respawn  ")
	buf.WriteString(colorMuted)
	buf.WriteString("Q")
	buf.WriteString(colorInk)
	buf.WriteString(" quit")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	buf.WriteString(colorMuted)
	buf.WriteString("  tip: eat pellets, avoid walls, and outlive the lobby")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	buf.WriteString(colorBorder)
	buf.WriteString("  ")
	buf.WriteString(strings.Repeat("─", fieldWidth+panelWidth+8))
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")
	buf.WriteString(colorMuted)
	buf.WriteString("  ready for another round")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")

	return buf.Bytes()
}

func renderArenaRow(state *GameState, dead map[string]bool, y int, overlayLine string, overlayCol int) string {
	var buf bytes.Buffer
	overlayStart := -1
	overlayCells := 0
	if overlayLine != "" {
		overlayStart = overlayCol / cellWidth
		overlayCells = len([]rune(overlayLine)) / cellWidth
	}

	for x := 0; x < Width; {
		if overlayStart >= 0 && x == overlayStart {
			buf.WriteString(colorOverlayBg + colorOverlay + overlayLine + colorReset)
			x += overlayCells
			continue
		}
		buf.WriteString(cellAt(state, dead, x, y))
		x++
	}
	return buf.String()
}

func cellAt(state *GameState, dead map[string]bool, x, y int) string {
	pt := Point{x, y}

	for nick, s := range state.Snakes {
		if dead[nick] || len(s.Body) == 0 {
			continue
		}
		color := snakeColor(nick)
		if s.Body[0].Equal(pt) {
			return colorArenaBg + color + cellHead + colorReset
		}
		for _, b := range s.Body[1:] {
			if b.Equal(pt) {
				return colorArenaBg + color + cellBody + colorReset
			}
		}
	}

	for _, a := range state.Apples {
		if a.Equal(pt) {
			return colorArenaBg + colorApple + cellApple + colorReset
		}
	}

	return colorArenaBg + cellEmpty + colorReset
}

func renderPanelLines(state *GameState, dead map[string]bool, order []string, overlayNick string) []string {
	type entry struct {
		nick  string
		score int
		dead  bool
	}

	entries := make([]entry, 0, len(state.Snakes))
	seen := make(map[string]bool, len(order))
	for _, nick := range order {
		s, ok := state.Snakes[nick]
		if !ok {
			continue
		}
		entries = append(entries, entry{nick: nick, score: scoreFor(state, nick, s), dead: dead[nick]})
		seen[nick] = true
	}
	for nick, s := range state.Snakes {
		if seen[nick] {
			continue
		}
		entries = append(entries, entry{nick: nick, score: scoreFor(state, nick, s), dead: dead[nick]})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].dead != entries[j].dead {
			return !entries[i].dead
		}
		if entries[i].score != entries[j].score {
			return entries[i].score > entries[j].score
		}
		return entries[i].nick < entries[j].nick
	})

	focusNick := overlayNick
	if focusNick == "" && len(order) > 0 {
		focusNick = order[0]
	}
	focusSnake := state.Snakes[focusNick]
	focusScore := 0
	focusLen := 0
	focusState := "SPECTATE"
	if focusSnake != nil {
		focusScore = scoreFor(state, focusNick, focusSnake)
		focusLen = focusSnake.Len()
		focusState = "ALIVE"
		if dead[focusNick] {
			focusState = "LOST"
		}
	}

	lines := []string{
		panelKV("MODE", "ARCADE"),
		panelKV("SCORE", fmt.Sprintf("%02d", focusScore)),
		panelKV("LENGTH", fmt.Sprintf("%02d", focusLen)),
		panelKV("PLAYERS", fmt.Sprintf("%02d", len(entries))),
		panelBlank(),
		panelKV("YOU", fallbackNick(focusNick)),
		panelKV("STATE", focusState),
		panelBlank(),
		panelLabel("RANKING"),
	}

	if len(entries) == 0 {
		lines = append(lines, panelText("waiting for players"))
	} else {
		for i, e := range entries {
			status := "OK"
			if e.dead {
				status = "KO"
			}
			label := fmt.Sprintf("%d %s %-9s %2d %s", i+1, nickHead(e.nick), trimRunes(e.nick, 9), e.score, status)
			lines = append(lines, panelText(label))
		}
	}

	for len(lines) < Height-3 {
		lines = append(lines, panelBlank())
	}
	lines = append(lines,
		panelLabel("CONTROLS"),
		panelText("WASD / arrows"),
		panelText("C respawn  Q quit"),
	)

	return lines
}

func renderPanelLine(lines []string, idx int) string {
	if idx >= 0 && idx < len(lines) {
		return colorPanelBg + colorInk + padPanel(lines[idx]) + colorReset
	}
	return colorPanelBg + strings.Repeat(" ", panelWidth) + colorReset
}

func panelKV(key, value string) string {
	value = trimRunes(value, 12)
	return fmt.Sprintf(" %-8s %16s ", key, value)
}

func panelLabel(label string) string {
	return fmt.Sprintf(" [%s]%s", label, strings.Repeat("-", max(0, panelWidth-len([]rune(label))-4)))
}

func panelText(text string) string {
	return " " + trimRunes(text, panelWidth-2)
}

func panelBlank() string {
	return ""
}

func padPanel(s string) string {
	runes := []rune(s)
	if len(runes) >= panelWidth {
		return string(runes[:panelWidth])
	}
	return s + strings.Repeat(" ", panelWidth-len(runes))
}

func fallbackNick(nick string) string {
	nick = strings.TrimSpace(nick)
	if nick == "" {
		return "-"
	}
	return trimRunes(nick, 12)
}

func deathOverlayLines(nick string) []string {
	if strings.TrimSpace(nick) == "" {
		return nil
	}

	name := trimRunes(nick, 12)
	if name == "" {
		name = "player"
	}
	nameLine := fmt.Sprintf("║   %-12s   ║", name)

	return []string{
		"╔══════════════════╗",
		"║   ROUND LOST     ║",
		nameLine,
		"║   press C        ║",
		"║   to respawn     ║",
		"╚══════════════════╝",
	}
}

func overlayOrigin(fieldWidth, overlayHeight, overlayWidth int) (int, int) {
	row := max(0, (Height-overlayHeight)/2)
	col := max(0, (fieldWidth-overlayWidth)/2)
	if col%cellWidth != 0 {
		col -= col % cellWidth
	}
	return row, col
}

func headCell(nick string) string {
	head := nickHead(nick)
	runes := []rune(head)
	if len(runes) == 0 {
		return cellHead
	}
	if len(runes) == 1 {
		return string(runes[0]) + "█"
	}
	return string(runes[:2])
}

func nickHead(nick string) string {
	runes := []rune(strings.ToUpper(strings.TrimSpace(nick)))
	if len(runes) == 0 {
		return "□"
	}
	r := runes[0]
	if r >= 'A' && r <= 'Z' {
		return string(rune(0x1F130 + (r - 'A')))
	}
	return string(r)
}

func snakeColor(nick string) string {
	sum := 0
	for _, r := range nick {
		sum += int(r)
	}
	return snakePalette[sum%len(snakePalette)]
}

func scoreFor(state *GameState, nick string, s *Snake) int {
	if score, ok := state.Scores[nick]; ok {
		return score
	}
	return s.Len() - 2
}

func trimRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
