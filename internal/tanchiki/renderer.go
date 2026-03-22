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

type tankDraw struct {
	glyph string
	color string
}

func renderFrame(g *Game, viewer string, order []string) []byte {
	var b bytes.Buffer
	fieldWidth := arenaW * cellW

	bulletMap := make(map[point]bool, len(g.bullets))
	for _, shot := range g.bullets {
		bulletMap[shot.pos] = true
	}
	tankMap := make(map[point]tankDraw, len(g.tanks))
	for _, t := range g.tanks {
		if !t.alive {
			continue
		}
		tankMap[t.pos] = tankDraw{glyph: tankGlyph(t.dir), color: tankColors[t.colorIdx%len(tankColors)]}
	}

	b.WriteString("\x1b[H")
	b.WriteString(cTitle + "  TANCHIKI" + cReset + cMuted + "  multiplayer tank arena" + cReset + "\r\n")
	b.WriteString(cBorder + "  +" + strings.Repeat("-", fieldWidth) + "+  +" + strings.Repeat("-", panelWidth) + "+\r\n" + cReset)

	panel := renderPanel(g, viewer, order)
	for y := 1; y <= arenaH; y++ {
		b.WriteString(cBorder + "  |" + cReset)
		for x := 1; x <= arenaW; x++ {
			p := point{x, y}
			cell, color := renderCell(g, p, bulletMap, tankMap)
			if color == "" {
				b.WriteString(cell)
			} else {
				b.WriteString(color)
				b.WriteString(cell)
				b.WriteString(cReset)
			}
		}
		b.WriteString(cBorder + "|  |" + cReset)
		if y-1 < len(panel) {
			b.WriteString(panel[y-1])
		} else {
			b.WriteString(strings.Repeat(" ", panelWidth))
		}
		b.WriteString(cBorder + "|\r\n" + cReset)
	}

	b.WriteString(cBorder + "  +" + strings.Repeat("-", fieldWidth) + "+  +" + strings.Repeat("-", panelWidth) + "+\r\n" + cReset)
	b.WriteString(cMuted + "  Arrows/WASD" + cText + " move  " + cMuted + "Space/F" + cText + " fire  " + cMuted + "Q" + cText + " quit\r\n" + cReset)
	b.WriteString(cMuted + "  #/=" + cText + " destructible walls  " + cMuted + "OO" + cText + " explosive barrels\r\n" + cReset)

	return b.Bytes()
}

func renderCell(g *Game, p point, bulletMap map[point]bool, tankMap map[point]tankDraw) (string, string) {
	if t, ok := tankMap[p]; ok {
		return t.glyph, t.color
	}
	if bulletMap[p] {
		return "..", cShot
	}
	if g.barrels[p] {
		return "OO", cBar
	}
	if hp, ok := g.walls[p]; ok {
		if hp >= 2 {
			return "##", cWall
		}
		return "==", cWall
	}
	return "  ", ""
}

func renderPanel(g *Game, viewer string, order []string) []string {
	lines := []string{
		pad(" MODE      BATTLE ARENA", panelWidth),
		pad(fmt.Sprintf(" PLAYERS   %d/6", len(g.players)), panelWidth),
		pad(fmt.Sprintf(" BULLETS   %d", len(g.bullets)), panelWidth),
		pad("", panelWidth),
		pad(" SCOREBOARD", panelWidth),
	}

	entries := make([]*tank, 0, len(order))
	for _, nick := range order {
		if t := g.tanks[nick]; t != nil {
			entries = append(entries, t)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].score != entries[j].score {
			return entries[i].score > entries[j].score
		}
		return entries[i].nick < entries[j].nick
	})

	for _, t := range entries {
		state := "ALIVE"
		if !t.alive {
			state = fmt.Sprintf("R%02d", t.respawnIn)
		}
		mark := " "
		if t.nick == viewer {
			mark = ">"
		}
		nick := pad(trimRunes(t.nick, 10), 10)
		line := fmt.Sprintf(" %s %s K:%02d D:%02d %s", mark, nick, t.score, t.deaths, state)
		line = tankColors[t.colorIdx%len(tankColors)] + line + cReset
		lines = append(lines, padANSI(line, panelWidth))
	}

	for len(lines) < arenaH {
		lines = append(lines, strings.Repeat(" ", panelWidth))
	}
	return lines
}

func tankGlyph(d dir) string {
	switch d {
	case dirUp:
		return "^^"
	case dirDown:
		return "vv"
	case dirLeft:
		return "<<"
	default:
		return ">>"
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
