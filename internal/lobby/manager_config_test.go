package lobby

import "testing"

func TestMaxPlayersForGame(t *testing.T) {
	if got := maxPlayersForGame(GameSnake); got != 8 {
		t.Fatalf("maxPlayersForGame(GameSnake) = %d, want 8", got)
	}
	if got := maxPlayersForGame(GamePingPong); got != 2 {
		t.Fatalf("maxPlayersForGame(GamePingPong) = %d, want 2", got)
	}
}
