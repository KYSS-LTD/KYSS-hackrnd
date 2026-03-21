package lobby

import "testing"

func TestMaxPlayersForGame(t *testing.T) {
	if got := maxPlayersForGame(GameSnake); got != 6 {
		t.Fatalf("maxPlayersForGame(GameSnake) = %d, want 6", got)
	}
	if got := maxPlayersForGame(GamePingPong); got != 2 {
		t.Fatalf("maxPlayersForGame(GamePingPong) = %d, want 2", got)
	}
}
