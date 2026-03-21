package snake

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	cellWidth  = 2
	panelWidth = 28

	cellEmpty = "  "
	cellBody  = "██"
	cellHead  = "▓▓"
	cellApple = "()"

	colorReset   = "\x1b[0m"
	colorBorder  = "\x1b[38;5;28m"
	colorInk     = "\x1b[38;5;250m"
	colorTitle   = "\x1b[1;38;5;120m"
	colorMuted   = "\x1b[38;5;108m"
	colorApple   = "\x1b[38;5;216m"
	colorOverlay = "\x1b[1;38;5;15m"
	colorDead    = "\x1b[1;38;5;203m"
)

type palettePair struct {
	Bright string
	Pale   string
}

var snakePalette = []palettePair{
	{Bright: "\x1b[1;38;5;51m", Pale: "\x1b[38;5;117m"},
	{Bright: "\x1b[1;38;5;226m", Pale: "\x1b[38;5;229m"},
	{Bright: "\x1b[1;38;5;46m", Pale: "\x1b[38;5;120m"},
	{Bright: "\x1b[1;38;5;213m", Pale: "\x1b[38;5;182m"},
	{Bright: "\x1b[1;38;5;208m", Pale: "\x1b[38;5;215m"},
	{Bright: "\x1b[1;38;5;39m", Pale: "\x1b[38;5;110m"},
}

func renderFrame(state *GameState, dead map[string]bool, order []string, colors map[string]int, viewerNick, overlayNick string) []byte {
	var buf bytes.Buffer
	fieldWidth := Width * cellWidth
	overlay := deathOverlayLines(overlayNick)
	overlayWidth := 0
	if len(overlay) > 0 {
		overlayWidth = len([]rune(overlay[0]))
	}
	overlayRow, overlayCol := overlayOrigin(fieldWidth, len(overlay), overlayWidth)
	panelLines := renderPanelLines(state, dead, order, colors, viewerNick)

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
		buf.WriteString(renderArenaRow(state, dead, colors, viewerNick, y, overlayLine, overlayCol))

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
	buf.WriteString("  ←↑↓→ / WASD / ЦФЫВ")
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

func renderArenaRow(state *GameState, dead map[string]bool, colors map[string]int, viewerNick string, y int, overlayLine string, overlayCol int) string {
	var buf bytes.Buffer
	overlayStart := -1
	overlayCells := 0
	if overlayLine != "" {
		overlayStart = overlayCol / cellWidth
		overlayCells = len([]rune(overlayLine)) / cellWidth
	}

	for x := 0; x < Width; {
		if overlayStart >= 0 && x == overlayStart {
			buf.WriteString(colorOverlay + overlayLine + colorReset)
			x += overlayCells
			continue
		}
		buf.WriteString(cellAt(state, dead, colors, viewerNick, x, y))
		x++
	}
	return buf.String()
}

func cellAt(state *GameState, dead map[string]bool, colors map[string]int, viewerNick string, x, y int) string {
	pt := Point{x, y}

	for nick, s := range state.Snakes {
		if dead[nick] || len(s.Body) == 0 {
			continue
		}
		color := snakeColor(colors, nick, viewerNick)
		if s.Body[0].Equal(pt) {
			return color + cellHead + colorReset
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

	return cellEmpty
}

func renderPanelLines(state *GameState, dead map[string]bool, order []string, colors map[string]int, viewerNick string) []string {
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

	focusNick := viewerNick
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
		panelKV("PLAYERS", fmt.Sprintf("%02d/06", len(entries))),
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
			status := colorInk + "OK" + colorReset
			if e.dead {
				status = colorDead + "KO" + colorReset
			}
			lines = append(lines, rankingLine(i+1, e.nick, e.score, status, snakeColor(colors, e.nick, viewerNick)))
		}
	}

	for len(lines) < Height-3 {
		lines = append(lines, panelBlank())
	}
	lines = append(lines,
		panelLabel("CONTROLS"),
		panelText("WASD / arrows / ЦФЫВ"),
		panelText("C respawn  Q quit"),
	)

	return lines
}

func renderPanelLine(lines []string, idx int) string {
	if idx >= 0 && idx < len(lines) {
		return padANSI(lines[idx], panelWidth)
	}
	return strings.Repeat(" ", panelWidth)
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

func rankingLine(rank int, nick string, score int, status string, nickColor string) string {
	head := nickHead(nick)
	plainNick := trimRunes(nick, 8)
	prefix := fmt.Sprintf(" %d %s ", rank, head)
	nickPart := nickColor + fmt.Sprintf("%-8s", plainNick) + colorReset
	scorePart := fmt.Sprintf(" %2d ", score)
	line := prefix + nickPart + scorePart + status
	return padANSI(line, panelWidth)
}

func panelBlank() string {
	return ""
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

func snakeColor(colors map[string]int, nick, viewerNick string) string {
	idx, ok := colors[nick]
	if !ok || idx < 0 || idx >= len(snakePalette) {
		idx = 0
	}
	pair := snakePalette[idx]
	if nick == viewerNick {
		return pair.Bright
	}
	return pair.Pale
}

func scoreFor(state *GameState, nick string, s *Snake) int {
	if score, ok := state.Scores[nick]; ok {
		return score
	}
	return max(0, s.Len()-2)
}

func trimRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func padANSI(s string, width int) string {
	visible := visibleRuneCount(s)
	if visible >= width {
		return truncateANSI(s, width)
	}
	return s + strings.Repeat(" ", width-visible)
}

func truncateANSI(s string, width int) string {
	var out strings.Builder
	visible := 0
	for i := 0; i < len(s) && visible < width; {
		if s[i] == 0x1b {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			out.WriteString(s[i:j])
			i = j
			continue
		}
		r := []rune(s[i:])[0]
		rLen := len(string(r))
		out.WriteRune(r)
		visible++
		i += rLen
	}
	out.WriteString(colorReset)
	return out.String()
}

func visibleRuneCount(s string) int {
	count := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		count++
		i += size
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
