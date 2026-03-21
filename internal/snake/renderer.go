package snake

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

const (
	cellWidth = 2

	cellEmpty = "· "
	cellBody  = "██"
	cellApple = "◉◉"

	colorReset     = "\x1b[0m"
	colorBorder    = "\x1b[38;5;45m"
	colorApple     = "\x1b[38;5;226m"
	colorTitle     = "\x1b[1;38;5;51m"
	colorSubtitle  = "\x1b[38;5;117m"
	colorMuted     = "\x1b[38;5;245m"
	colorDead      = "\x1b[38;5;203m"
	colorOverlay   = "\x1b[1;38;5;198m"
	colorOverlayBg = "\x1b[48;5;53m"
)

var snakePalette = []string{
	"\x1b[38;5;81m",
	"\x1b[38;5;118m",
	"\x1b[38;5;213m",
	"\x1b[38;5;208m",
	"\x1b[38;5;159m",
	"\x1b[38;5;154m",
	"\x1b[38;5;220m",
	"\x1b[38;5;141m",
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

	buf.WriteString("\x1b[H")
	buf.WriteString(colorTitle)
	buf.WriteString("  ╭─ SNAKE ARENA ─")
	buf.WriteString(strings.Repeat("─", max(0, fieldWidth-15)))
	buf.WriteString("╮\r\n")
	buf.WriteString(colorMuted)
	buf.WriteString("  │ ")
	buf.WriteString(colorSubtitle)
	buf.WriteString("лови ◉◉, расти и переживи остальных")
	buf.WriteString(colorMuted)
	buf.WriteString(strings.Repeat(" ", max(0, fieldWidth-35)))
	buf.WriteString("│\r\n")

	buf.WriteString(colorBorder)
	buf.WriteString("  ╔")
	for i := 0; i < fieldWidth; i++ {
		buf.WriteString("═")
	}
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
		buf.WriteString("║")
		buf.WriteString(colorReset)
		buf.WriteString("\r\n")
	}

	buf.WriteString(colorBorder)
	buf.WriteString("  ╚")
	for i := 0; i < fieldWidth; i++ {
		buf.WriteString("═")
	}
	buf.WriteString("╝")
	buf.WriteString(colorReset)
	buf.WriteString("\r\n")

	renderLeaderboard(&buf, state, dead, order)

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
			return color + headCell(nick) + colorReset
		}
		for _, b := range s.Body[1:] {
			if b.Equal(pt) {
				return color + cellBody + colorReset
			}
		}
	}

	for _, a := range state.Apples {
		if a.Equal(pt) {
			return colorApple + cellApple + colorReset
		}
	}

	return colorMuted + cellEmpty + colorReset
}

func renderLeaderboard(buf *bytes.Buffer, state *GameState, dead map[string]bool, order []string) {
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

	lineW := Width*cellWidth + 4
	buf.WriteString(colorBorder + "  " + strings.Repeat("─", lineW) + colorReset + "\r\n")
	buf.WriteString(colorTitle + "  ◇ PILOTS" + colorReset + "\r\n")

	for i, e := range entries {
		status := colorMuted + "ALIVE" + colorReset
		if e.dead {
			status = colorDead + "DEAD " + colorReset
		}
		nick := trimRunes(e.nick, 12)
		line := fmt.Sprintf("  %d. %s%s %-12s%s  %s%2d%s  %s\r\n",
			i+1,
			snakeColor(e.nick),
			nickHead(e.nick),
			nick,
			colorReset,
			colorApple,
			e.score,
			colorReset,
			status,
		)
		buf.WriteString(line)
	}

	if len(entries) == 0 {
		buf.WriteString(colorMuted + "  Никого на арене — заходи первым.\r\n" + colorReset)
	}

	buf.WriteString(colorBorder + "  " + strings.Repeat("─", lineW) + colorReset + "\r\n")
	buf.WriteString(colorSubtitle + "  WASD/стрелки" + colorReset)
	buf.WriteString(colorMuted + " — движение   " + colorReset)
	buf.WriteString(colorSubtitle + "C" + colorReset)
	buf.WriteString(colorMuted + " — респавн   " + colorReset)
	buf.WriteString(colorSubtitle + "Q" + colorReset)
	buf.WriteString(colorMuted + " — выход\r\n" + colorReset)
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
		return "[]"
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
