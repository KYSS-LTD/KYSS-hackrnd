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

func renderBase(players map[string]*Player, apples []Point) []byte {
	var buf bytes.Buffer

	buf.WriteString("\x1b[H")

	buf.WriteString("╔")
	for i := 0; i < FieldWidth; i++ {
		buf.WriteString("══")
	}
	buf.WriteString("╗\r\n")

	for y := 0; y < FieldHeight; y++ {
		buf.WriteString("║")
		for x := 0; x < FieldWidth; x++ {
			buf.WriteString(cellAt(players, apples, x, y))
		}
		buf.WriteString("║\r\n")
	}

	buf.WriteString("╚")
	for i := 0; i < FieldWidth; i++ {
		buf.WriteString("══")
	}
	buf.WriteString("╝\r\n")

	renderLeaderboard(&buf, players)

	return buf.Bytes()
}

func cellAt(players map[string]*Player, apples []Point, x, y int) string {
	pt := Point{x, y}

	for _, p := range players {
		if !p.IsAlive() {
			continue
		}
		if p.Snake.Body[0].Equal(pt) {
			return nickHead(p.Nick)
		}
		for _, b := range p.Snake.Body[1:] {
			if b.Equal(pt) {
				return cellBody
			}
		}
	}

	for _, a := range apples {
		if a.Equal(pt) {
			return cellApple
		}
	}

	return cellEmpty
}

func renderLeaderboard(buf *bytes.Buffer, players map[string]*Player) {
	type entry struct {
		nick  string
		score int
		dead  bool
	}

	var entries []entry
	for nick, p := range players {
		entries = append(entries, entry{
			nick:  nick,
			score: p.Snake.Len(),
			dead:  !p.IsAlive(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].dead != entries[j].dead {
			return !entries[i].dead
		}
		return entries[i].score > entries[j].score
	})

	lineW := FieldWidth*2 + 2
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
		nick := e.nick
		if len([]rune(nick)) > 10 {
			nick = string([]rune(nick)[:10])
		}
		line := fmt.Sprintf("  %d. %-10s  %s%-3d%s\r\n", i+1, nick, bar, e.score, deadMark)
		buf.WriteString(line)
	}

	if len(entries) == 0 {
		buf.WriteString("  No players yet.\r\n")
	}

	buf.WriteString(strings.Repeat("─", lineW) + "\r\n")
	buf.WriteString("  WASD: move   C: respawn   Q: quit\r\n")
}

func renderDeathOverlay() []byte {
	var buf bytes.Buffer

	col := 9
	row := 4

	lines := []string{
		"╔══════════════════╗",
		"║    YOU  DIED!    ║",
		"║  Press C to      ║",
		"║    respawn       ║",
		"╚══════════════════╝",
	}

	for i, line := range lines {
		buf.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", row+i, col, line))
	}

	return buf.Bytes()
}
