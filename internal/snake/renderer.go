package snake

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

const (
	cellEmpty = "  "
	cellBody  = "██"
	cellApple = "()"
)

func renderFrame(state *GameState, dead map[string]bool, order []string) []byte {
	var buf bytes.Buffer

	buf.WriteString("\x1b[H")

	buf.WriteString("╔")
	for i := 0; i < Width; i++ {
		buf.WriteString("══")
	}
	buf.WriteString("╗\r\n")

	for y := 0; y < Height; y++ {
		buf.WriteString("║")
		for x := 0; x < Width; x++ {
			buf.WriteString(cellAt(state, dead, x, y))
		}
		buf.WriteString("║\r\n")
	}

	buf.WriteString("╚")
	for i := 0; i < Width; i++ {
		buf.WriteString("══")
	}
	buf.WriteString("╝\r\n")

	renderLeaderboard(&buf, state, dead, order)

	return buf.Bytes()
}

func cellAt(state *GameState, dead map[string]bool, x, y int) string {
	pt := Point{x, y}

	for nick, s := range state.Snakes {
		if dead[nick] || len(s.Body) == 0 {
			continue
		}
		if s.Body[0].Equal(pt) {
			return nickHead(nick)
		}
		for _, b := range s.Body[1:] {
			if b.Equal(pt) {
				return cellBody
			}
		}
	}

	for _, a := range state.Apples {
		if a.Equal(pt) {
			return cellApple
		}
	}

	return cellEmpty
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

	lineW := Width*2 + 2
	buf.WriteString(strings.Repeat("─", lineW) + "\r\n")
	buf.WriteString("  LEADERBOARD\r\n")

	for i, e := range entries {
		bar := strings.Repeat("█", e.score)
		if len(bar) > 20 {
			bar = bar[:20]
		}
		deadMark := ""
		if e.dead {
			deadMark = "  [DEAD]"
		}
		nick := trimRunes(e.nick, 10)
		line := fmt.Sprintf("  %d. %-10s  %s%-3d%s\r\n", i+1, nick, bar, e.score, deadMark)
		buf.WriteString(line)
	}

	if len(entries) == 0 {
		buf.WriteString("  No players yet.\r\n")
	}

	buf.WriteString(strings.Repeat("─", lineW) + "\r\n")
	buf.WriteString("  WASD: move   C: respawn   Q: quit\r\n")
}

func renderDeathOverlay(nick string) []byte {
	var buf bytes.Buffer

	col := 7
	row := 4
	name := trimRunes(nick, 12)
	if name == "" {
		name = "player"
	}
	nameLine := fmt.Sprintf("║ %-16s ║", name)

	lines := []string{
		"╔══════════════════╗",
		"║    YOU  DIED!    ║",
		nameLine,
		"║  Press C to      ║",
		"║    respawn       ║",
		"╚══════════════════╝",
	}

	for i, line := range lines {
		buf.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", row+i, col, line))
	}

	return buf.Bytes()
}

func nickHead(nick string) string {
	runes := []rune(strings.ToUpper(strings.TrimSpace(nick)))
	if len(runes) == 0 {
		return "[]"
	}
	return fmt.Sprintf("[%c]", runes[0])
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
