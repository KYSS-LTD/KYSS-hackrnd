package hub

import "testing"

func TestAvailableGamesIncludesPingPong(t *testing.T) {
	games := availableGames()
	foundPingPong := false
	for _, game := range games {
		if game.id == "pingpong" {
			foundPingPong = true
			break
		}
	}
	if !foundPingPong {
		t.Fatalf("availableGames() missing pingpong: %#v", games)
	}
}
