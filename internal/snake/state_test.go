package snake

import "testing"

func TestTickClearsDeadSnakeBody(t *testing.T) {
	state := newGameState()
	state.Snakes["dead"] = &Snake{
		Body:    []Point{{0, 0}, {1, 0}},
		Dir:     DirLeft,
		NextDir: DirLeft,
	}
	state.Scores["dead"] = 2

	dead := state.tick()
	if !dead["dead"] {
		t.Fatalf("expected snake to die on wall collision")
	}
	if got := len(state.Snakes["dead"].Body); got != 0 {
		t.Fatalf("dead snake body length = %d, want 0", got)
	}
}
