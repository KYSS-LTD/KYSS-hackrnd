package hub

import (
	"strings"
	"testing"
)

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

func TestRenderHeaderHandlesNarrowWidths(t *testing.T) {
	for _, width := range []int{0, 1, 2} {
		header := renderHeader(width, "player")
		if !strings.Contains(header, "SSH GAMES") {
			t.Fatalf("renderHeader(%d) missing title: %q", width, header)
		}
		if !strings.Contains(header, "player: player") {
			t.Fatalf("renderHeader(%d) missing player line: %q", width, header)
		}
	}
}

func TestAvailableGamesIncludesKitchen(t *testing.T) {
	games := availableGames()
	foundKitchen := false
	for _, game := range games {
		if game.id == "kitchen" {
			foundKitchen = true
			break
		}
	}
	if !foundKitchen {
		t.Fatalf("availableGames() missing kitchen: %#v", games)
	}
}
