package snake

import "testing"

func TestJoinRejectsDuplicateNick(t *testing.T) {
	game := NewGame()
	first := newPlayer("alice", discardWriter{})
	second := newPlayer("alice", discardWriter{})
	defer first.close()
	defer second.close()

	if err := game.Join("alice", first); err != nil {
		t.Fatalf("first join failed: %v", err)
	}
	defer game.Leave("alice")

	if err := game.Join("alice", second); err == nil {
		t.Fatalf("duplicate nick should be rejected")
	}
}

func TestJoinRejectsSeventhPlayer(t *testing.T) {
	game := NewGame()
	players := make([]*Player, 0, maxSnakePlayers)
	for i := 0; i < maxSnakePlayers; i++ {
		nick := string(rune('a' + i))
		p := newPlayer(nick, discardWriter{})
		players = append(players, p)
		if err := game.Join(nick, p); err != nil {
			t.Fatalf("join %q failed: %v", nick, err)
		}
		defer game.Leave(nick)
	}

	extra := newPlayer("z", discardWriter{})
	defer extra.close()
	if err := game.Join("z", extra); err == nil {
		t.Fatalf("seventh player should be rejected")
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
