package hub

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"log"

	gossh "github.com/gliderlabs/ssh"
	cryptossh "golang.org/x/crypto/ssh"

	"ssh-games/internal/lobby"
	"ssh-games/internal/tui"
)

type Server struct {
	manager  *lobby.Manager
	registry *lobby.Registry
	addr     string
}

func NewServer(addr string, manager *lobby.Manager, registry *lobby.Registry) *Server {
	return &Server{
		addr:     addr,
		manager:  manager,
		registry: registry,
	}
}

func (s *Server) ListenAndServe() error {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate host key: %w", err)
	}
	signer, err := cryptossh.NewSignerFromKey(privKey)
	if err != nil {
		return fmt.Errorf("create signer: %w", err)
	}

	srv := &gossh.Server{
		Addr: s.addr,
		Handler: func(sess gossh.Session) {
			s.handleSession(sess)
		},
		PasswordHandler: func(_ gossh.Context, _ string) bool {
			return true
		},
		PublicKeyHandler: func(_ gossh.Context, _ gossh.PublicKey) bool {
			return true
		},
		PtyCallback: func(_ gossh.Context, _ gossh.Pty) bool {
			return true
		},
		HostSigners: []gossh.Signer{signer},
	}

	log.Printf("Hub SSH server listening on %s", s.addr)
	return srv.ListenAndServe()
}

func (s *Server) handleSession(sess gossh.Session) {
	_, _, isPty := sess.Pty()
	if !isPty {
		fmt.Fprintln(sess, "Error: PTY required. Connect with: ssh -t -p 2222 player@<host>")
		return
	}

	defer func() {
		fmt.Fprint(sess, tui.NormalScreen+tui.ShowCursor)
	}()

	for {
		nick, err := runNickInput(sess)
		if err != nil {
			return
		}

		gameType, err := runGameMenu(sess)
		if err != nil {
			return
		}

		lobbyID, err := runLobbyList(sess, s.registry, s.manager, gameType)
		if err != nil {
			return
		}

		if lobbyID == -1 {
			continue
		}

		var addr string

		if lobbyID == 0 {
			fmt.Fprint(sess, tui.ClearScreen)
			tui.WriteAt(sess, 3, 3, "  Creating new lobby, please wait...")

			newLobby, createErr := s.manager.CreateLobby(gameType)
			if createErr != nil {
				log.Printf("create lobby error: %v", createErr)
				tui.WriteAt(sess, 4, 3, fmt.Sprintf("  Error: %v", createErr))
				tui.WriteAt(sess, 5, 3, "  Press any key to continue...")
				buf := make([]byte, 1)
				sess.Read(buf)
				continue
			}
			addr = newLobby.Addr
			lobbyID = newLobby.ID
		} else {
			l, ok := s.registry.Get(lobbyID)
			if !ok {
				continue
			}
			addr = l.Addr
		}

		s.registry.IncrPlayerCount(lobbyID)

		proxyErr := proxyToGame(sess, addr, nick)

		s.registry.DecrPlayerCount(lobbyID)

		if proxyErr != nil && proxyErr != io.EOF {
			log.Printf("proxy session ended: %v", proxyErr)
		}

		fmt.Fprint(sess, tui.NormalScreen+tui.ShowCursor+tui.ClearScreen)
		tui.WriteAt(sess, 3, 3, "  Session ended. Press any key to return to menu...")
		buf := make([]byte, 1)
		sess.Read(buf)
	}
}
