package snake

import (
	"strings"
	"testing"
)

func TestNickHeadUsesSquaredLatinLetters(t *testing.T) {
	cases := map[string]string{
		"snake": "🅂",
		"alpha": "🄰",
		"":      "□",
		"7up":   "7",
	}

	for nick, want := range cases {
		if got := nickHead(nick); got != want {
			t.Fatalf("nickHead(%q) = %q, want %q", nick, got, want)
		}
	}
}

func TestRenderFrameUsesExpandedArena(t *testing.T) {
	state := newGameState()
	state.Snakes["snake"] = &Snake{
		Body:    []Point{{5, 5}, {4, 5}},
		Dir:     DirRight,
		NextDir: DirRight,
	}
	state.Apples = []Point{{10, 10}}
	state.Scores["snake"] = 3

	frame := string(renderFrame(state, map[string]bool{}, []string{"snake"}, map[string]int{"snake": 0}, "snake", ""))

	if count := strings.Count(frame, "\r\n"); count < Height+8 {
		t.Fatalf("frame should have at least %d lines, got %d", Height+8, count)
	}
	if !strings.Contains(frame, strings.Repeat("═", Width*cellWidth)) {
		t.Fatalf("frame should contain top border sized to width %d", Width*cellWidth)
	}
	if !strings.Contains(frame, "ЦФЫВ") {
		t.Fatalf("frame should advertise Cyrillic WASD controls")
	}
}

func TestRenderFrameEmbedsDeathOverlay(t *testing.T) {
	frame := string(renderFrame(newGameState(), map[string]bool{"snake": true}, []string{"snake"}, map[string]int{"snake": 0}, "snake", "snake"))

	if !strings.Contains(frame, "ROUND LOST") {
		t.Fatalf("frame should contain death overlay")
	}
}

func TestSnakeColorUsesBrightForViewerAndPaleForOthers(t *testing.T) {
	colors := map[string]int{"alice": 1, "bob": 1}
	if got := snakeColor(colors, "alice", "alice"); got != snakePalette[1].Bright {
		t.Fatalf("viewer color = %q, want %q", got, snakePalette[1].Bright)
	}
	if got := snakeColor(colors, "bob", "alice"); got != snakePalette[1].Pale {
		t.Fatalf("enemy color = %q, want %q", got, snakePalette[1].Pale)
	}
}
