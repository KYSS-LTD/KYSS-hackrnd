package snake

import (
	"math/rand"
	"testing"
)

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

func TestAddPlayerSpawnsWithFreeCellsAhead(t *testing.T) {
	rand.Seed(1)

	state := newGameState()
	state.addPlayer("alice")

	snake := state.Snakes["alice"]
	if snake == nil {
		t.Fatalf("snake was not created")
	}
	if len(snake.Body) != 2 {
		t.Fatalf("snake body length = %d, want 2", len(snake.Body))
	}

	head := snake.Head()
	tail := snake.Body[1]
	expectedTail := state.spawnTail(head, snake.Dir)
	if !tail.Equal(expectedTail) {
		t.Fatalf("tail = %+v, want %+v", tail, expectedTail)
	}

	for step := 1; step <= minSpawnFreeAhead; step++ {
		ahead := Point{X: head.X + snake.Dir.X*step, Y: head.Y + snake.Dir.Y*step}
		if !state.isFreeAt(ahead) {
			t.Fatalf("cell %+v ahead of spawn is not free", ahead)
		}
	}
}

func TestSpawnCandidatesIncludeMultipleDirections(t *testing.T) {
	state := newGameState()
	state.Apples = []Point{{0, 0}}

	candidates := state.spawnCandidates()
	seen := make(map[Point]bool)
	for _, candidate := range candidates {
		seen[candidate.Dir] = true
	}

	for _, dir := range spawnDirections {
		if !seen[dir] {
			t.Fatalf("direction %+v was not available for spawn", dir)
		}
	}
}
