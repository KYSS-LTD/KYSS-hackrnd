package finalsentence

import (
	"testing"
	"time"
)

func TestProgressRatioAcrossLines(t *testing.T) {
	prog := &playerProgress{lineIndex: 1, typed: []typedRune{{value: 'g', correct: true}, {value: 'o', correct: true}}}
	lines := []string{"abc", "go", "z"}
	got := progressRatioForLines(lines, prog)
	want := float64(5) / float64(6)
	if got != want {
		t.Fatalf("progressRatioForLines() = %v, want %v", got, want)
	}
}

func TestHandleInputCompletesRound(t *testing.T) {
	g := &Game{progress: map[string]*playerProgress{"ada": {}}, roundIndex: 0, roundStart: time.Now()}
	for _, line := range g.currentLines() {
		for _, r := range line {
			g.handleInput("ada", string(r), time.Now())
		}
	}
	prog := g.progress["ada"]
	if !prog.finished {
		t.Fatalf("player should finish all lines")
	}
}
