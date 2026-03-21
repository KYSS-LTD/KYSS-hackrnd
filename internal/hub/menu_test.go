package hub

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"ssh-games/internal/lobby"

	gossh "github.com/gliderlabs/ssh"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestAvailableGamesIncludesPingPongAndFinalSentence(t *testing.T) {
	games := availableGames()
	foundPingPong := false
	foundFinalSentence := false
	for _, game := range games {
		if game.id == "pingpong" {
			foundPingPong = true
		}
		if game.id == "finalsentence" {
			foundFinalSentence = true
		}
	}
	if !foundPingPong || !foundFinalSentence {
		t.Fatalf("availableGames() missing entries: %#v", games)
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

func TestSelectLobbyCreateUsesCurrentGame(t *testing.T) {
	sess := &stubSession{reads: [][]byte{{13}}}
	mgr := &lobby.Manager{}
	connectedLobbyID := ""
	createdGames := make([]lobby.GameType, 0, 1)

	menu := &menu{
		sess:    sess,
		manager: mgr,
		nick:    "tester",
		w:       80,
		h:       24,
		createLobby: func(game lobby.GameType) (*lobby.Lobby, error) {
			createdGames = append(createdGames, game)
			return &lobby.Lobby{ID: "pingpong-1", Game: game}, nil
		},
		connectLobby: func(lobbyID string) {
			connectedLobbyID = lobbyID
		},
	}

	action := menu.selectLobby(lobby.GamePingPong)
	if action != "back" {
		t.Fatalf("selectLobby() action = %q, want back", action)
	}
	if len(createdGames) != 1 {
		t.Fatalf("createLobby called %d times, want 1", len(createdGames))
	}
	if createdGames[0] != lobby.GamePingPong {
		t.Fatalf("createLobby called with %q, want %q", createdGames[0], lobby.GamePingPong)
	}
	if connectedLobbyID != "pingpong-1" {
		t.Fatalf("connectLobby() = %q, want pingpong-1", connectedLobbyID)
	}
}

type stubSession struct {
	reads [][]byte
	readI int
	bytes.Buffer
}

func (s *stubSession) Read(p []byte) (int, error) {
	if s.readI >= len(s.reads) {
		return 0, io.EOF
	}
	n := copy(p, s.reads[s.readI])
	s.readI++
	return n, nil
}

func (s *stubSession) User() string               { return "tester" }
func (s *stubSession) RemoteAddr() net.Addr       { return stubAddr("remote") }
func (s *stubSession) LocalAddr() net.Addr        { return stubAddr("local") }
func (s *stubSession) Environ() []string          { return nil }
func (s *stubSession) Exit(int) error             { return nil }
func (s *stubSession) Command() []string          { return nil }
func (s *stubSession) RawCommand() string         { return "" }
func (s *stubSession) Subsystem() string          { return "" }
func (s *stubSession) PublicKey() gossh.PublicKey { return nil }
func (s *stubSession) Context() gossh.Context     { return &stubContext{Context: context.Background()} }
func (s *stubSession) Permissions() gossh.Permissions {
	return gossh.Permissions{Permissions: &cryptossh.Permissions{}}
}
func (s *stubSession) Pty() (gossh.Pty, <-chan gossh.Window, bool)    { return gossh.Pty{}, nil, false }
func (s *stubSession) Signals(chan<- gossh.Signal)                    {}
func (s *stubSession) Break(chan<- bool)                              {}
func (s *stubSession) Close() error                                   { return nil }
func (s *stubSession) CloseWrite() error                              { return nil }
func (s *stubSession) SendRequest(string, bool, []byte) (bool, error) { return false, nil }
func (s *stubSession) Stderr() io.ReadWriter                          { return &bytes.Buffer{} }

type stubAddr string

func (a stubAddr) Network() string { return string(a) }
func (a stubAddr) String() string  { return string(a) }

type stubContext struct{ context.Context }

func (*stubContext) Lock()                 {}
func (*stubContext) Unlock()               {}
func (*stubContext) User() string          { return "tester" }
func (*stubContext) SessionID() string     { return "session" }
func (*stubContext) ClientVersion() string { return "client" }
func (*stubContext) ServerVersion() string { return "server" }
func (*stubContext) RemoteAddr() net.Addr  { return stubAddr("remote") }
func (*stubContext) LocalAddr() net.Addr   { return stubAddr("local") }
func (*stubContext) Permissions() *gossh.Permissions {
	return &gossh.Permissions{Permissions: &cryptossh.Permissions{}}
}
func (*stubContext) SetValue(interface{}, interface{}) {}
