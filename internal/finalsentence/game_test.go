package finalsentence

import (
	"strings"
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

func TestHandleInputBlocksAfterMistake(t *testing.T) {
	now := time.Now()
	g := &Game{progress: map[string]*playerProgress{"ada": {}}, roundIndex: 0, roundStart: now}

	g.handleInput("ada", "x", now)
	prog := g.progress["ada"]
	if len(prog.typed) != 0 {
		t.Fatalf("wrong input should not advance progress")
	}
	if prog.mistake == nil || prog.mistake.value != 'x' || prog.mistake.index != 0 {
		t.Fatalf("wrong input should be stored as a visible mistake")
	}
	if !prog.blockedUntil.Equal(now.Add(inputLockTime)) {
		t.Fatalf("blockedUntil = %v, want %v", prog.blockedUntil, now.Add(inputLockTime))
	}

	g.handleInput("ada", string([]rune(g.currentLines()[0])[0]), now.Add(time.Second))
	if len(prog.typed) != 0 {
		t.Fatalf("input during lock should be ignored")
	}

	g.handleInput("ada", string([]rune(g.currentLines()[0])[0]), now.Add(inputLockTime))
	if len(prog.typed) != 1 {
		t.Fatalf("correct input after lock should advance progress")
	}
	if prog.mistake != nil {
		t.Fatalf("mistake should be cleared after correct retry")
	}
}

func TestRenderActiveLineShowsLockAndRetryStates(t *testing.T) {
	line := "ab"
	locked := renderActiveLine(line, playerSnapshot{
		Mistake:      &mistakeState{value: 'x', index: 0},
		InputBlocked: true,
	}, 4)
	if !strings.Contains(locked, colorRed+"x") {
		t.Fatalf("locked state should render mistake in red: %q", locked)
	}
	if !strings.Contains(locked, colorDim+"b") {
		t.Fatalf("locked state should dim upcoming text: %q", locked)
	}

	unlocked := renderActiveLine(line, playerSnapshot{
		Mistake: &mistakeState{value: 'x', index: 0},
	}, 4)
	if !strings.Contains(unlocked, colorBlue+"x") {
		t.Fatalf("unlocked state should render mistake in blue: %q", unlocked)
	}
	if !strings.Contains(unlocked, colorWhite+"b") {
		t.Fatalf("unlocked state should restore white text: %q", unlocked)
	}
}
