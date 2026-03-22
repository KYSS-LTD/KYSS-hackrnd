package tanchiki

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

var tankColors = []string{
	"\x1b[1;38;5;51m",
	"\x1b[1;38;5;226m",
	"\x1b[1;38;5;208m",
	"\x1b[1;38;5;213m",
	"\x1b[1;38;5;46m",
	"\x1b[1;38;5;39m",
}

const (
	cellW      = 2
	panelWidth = 34

	cReset  = "\x1b[0m"
	cBorder = "\x1b[38;5;28m"
	cTitle  = "\x1b[1;38;5;121m"
	cText   = "\x1b[38;5;250m"
	cMuted  = "\x1b[38;5;108m"
	cWall   = "\x1b[38;5;180m"
	cBar    = "\x1b[1;38;5;203m"
	cShot   = "\x1b[1;38;5;15m"
)

func renderFrame(g *Game, viewer string, order []string) []byte {
	var b bytes.Buffer
	fieldWidth := arenaW * cellW

	b.WriteString("\x1b[H")
	b.WriteString(cTitle + "  TANCHIKI" + cReset + cMuted + "  терминальные танки" + cReset + "\r\n")
	b.WriteString(cBorder + "  ╔" + strings.Repeat("═", fieldWidth) + "╗  ╔" + strings.Repeat("═", panelWidth) + "╗\r\n" + cReset)

	panel := renderPanel(g, viewer, order)
	for y := 1; y <= arenaH; y++ {
		b.WriteString(cBorder + "  ║" + cReset)
		for x := 1; x <= arenaW; x++ {
			b.WriteString(renderCell(g, point{x, y}))
		}
		b.WriteString(cBorder + "║  ║" + cReset)
		if y-1 < len(panel) {
			b.WriteString(panel[y-1])
		} else {
			b.WriteString(strings.Repeat(" ", panelWidth))
		}
		b.WriteString(cBorder + "║\r\n" + cReset)
	}

	b.WriteString(cBorder + "  ╚" + strings.Repeat("═", fieldWidth) + "╝  ╚" + strings.Repeat("═", panelWidth) + "╝\r\n" + cReset)
	b.WriteString(cMuted + "  WASD/стрелки" + cText + " движение  " + cMuted + "Space/F" + cText + " выстрел  " + cMuted + "Q" + cText + " выход\r\n" + cReset)
	b.WriteString(cMuted + "  стены ломаются по HP, бочки взрываются и убивают в радиусе 1 клетки\r\n" + cReset)

	return b.Bytes()
}

func renderCell(g *Game, p point) string {
	cell := "  "
	color := ""

	if hp, ok := g.walls[p]; ok {
		color = cWall
		if hp >= 2 {
			cell = "▓▓"
		} else {
			cell = "▒▒"
		}
	}
	if g.barrels[p] {
		color = cBar
		cell = "¤¤"
	}
	for _, shot := range g.bullets {
		if shot.pos == p {
			color = cShot
			cell = "••"
		}
	}
	for _, t := range g.tanks {
		if t.alive && t.pos == p {
			color = tankColors[t.colorIdx%len(tankColors)]
			cell = tankGlyph(t.dir)
		}
	}

	if color == "" {
		return cell
	}
	return color + cell + cReset
}

func renderPanel(g *Game, viewer string, order []string) []string {
	lines := []string{
		pad(" MODE      BATTLE ARENA", panelWidth),
		pad(fmt.Sprintf(" PLAYERS   %d/6", len(g.players)), panelWidth),
		pad(fmt.Sprintf(" BULLETS   %d", len(g.bullets)), panelWidth),
		pad("", panelWidth),
		pad(" SCOREBOARD", panelWidth),
	}

	entries := make([]string, 0, len(order))
	for _, nick := range order {
		t := g.tanks[nick]
		if t == nil {
			continue
		}
		state := "ALIVE"
		if !t.alive {
			state = fmt.Sprintf("RSP %02d", t.respawnIn)
		}
		mark := " "
		if nick == viewer {
			mark = ">"
		}
		coloredNick := tankColors[t.colorIdx%len(tankColors)] + trimRunes(nick, 10) + cReset
		entries = append(entries, fmt.Sprintf(" %s %-12s K:%02d D:%02d %-6s", mark, coloredNick, t.score, t.deaths, state))
	}
	sort.Strings(entries)
	for _, e := range entries {
		lines = append(lines, padANSI(e, panelWidth))
	}

	for len(lines) < arenaH {
		lines = append(lines, strings.Repeat(" ", panelWidth))
	}
	return lines
}

func tankGlyph(d dir) string {
	switch d {
	case dirUp:
		return "▲▲"
	case dirDown:
		return "▼▼"
	case dirLeft:
		return "◀◀"
	default:
		return "▶▶"
	}
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func padANSI(s string, width int) string {
	visible := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		visible++
	}
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}
