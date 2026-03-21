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
	cReset = "\x1b[0m"
	cEdge  = "\x1b[38;5;242m"
	cHUD   = "\x1b[38;5;250m"
	cWall  = "\x1b[38;5;180m"
	cBar   = "\x1b[1;38;5;203m"
	cShot  = "\x1b[1;38;5;15m"
)

func renderFrame(g *Game, viewer string, order []string) []byte {
	var b bytes.Buffer
	b.WriteString("\x1b[H")
	b.WriteString("\x1b[2J")
	b.WriteString("\x1b[H")
	b.WriteString("\x1b[1;38;5;120m  TANCHIKI  \x1b[0m")
	b.WriteString("\x1b[38;5;245mмультиплеерные танки\x1b[0m\r\n")

	b.WriteString(cEdge + "  ╔" + strings.Repeat("═", arenaW) + "╗\r\n" + cReset)
	for y := 1; y <= arenaH; y++ {
		b.WriteString(cEdge + "  ║" + cReset)
		for x := 1; x <= arenaW; x++ {
			p := point{x, y}
			cell := " "
			color := ""
			if hp, ok := g.walls[p]; ok {
				if hp == 2 {
					cell = "▓"
				} else {
					cell = "▒"
				}
				color = cWall
			}
			if g.barrels[p] {
				cell = "¤"
				color = cBar
			}
			for _, shot := range g.bullets {
				if shot.pos == p {
					cell = "•"
					color = cShot
				}
			}
			for _, t := range g.tanks {
				if t.alive && t.pos == p {
					cell = tankRune(t.dir)
					color = tankColors[t.colorIdx%len(tankColors)]
				}
			}
			if color == "" {
				b.WriteString(cell)
			} else {
				b.WriteString(color + cell + cReset)
			}
		}
		b.WriteString(cEdge + "║" + cReset + "\r\n")
	}
	b.WriteString(cEdge + "  ╚" + strings.Repeat("═", arenaW) + "╝\r\n" + cReset)

	b.WriteString(cHUD + "  управление: WASD/стрелки — движение, Space/F — выстрел, Q — выход\r\n" + cReset)
	b.WriteString(cHUD + "  стены разрушаются по кусочкам, бочки взрываются цепочкой\r\n" + cReset)

	table := make([]string, 0, len(order))
	for _, nick := range order {
		t := g.tanks[nick]
		if t == nil {
			continue
		}
		state := "alive"
		if !t.alive {
			state = fmt.Sprintf("respawn %d", t.respawnIn)
		}
		mark := " "
		if nick == viewer {
			mark = ">"
		}
		line := fmt.Sprintf("%s %-12s  K:%2d  D:%2d  %s", mark, nick, t.score, t.deaths, state)
		table = append(table, line)
	}
	sort.Strings(table)
	for _, line := range table {
		b.WriteString("  " + line + "\r\n")
	}
	return b.Bytes()
}

func tankRune(d dir) string {
	switch d {
	case dirUp:
		return "▲"
	case dirDown:
		return "▼"
	case dirLeft:
		return "◀"
	default:
		return "▶"
	}
}
