package hub

import (
	"fmt"
	"io"
	"log"

	gossh "github.com/gliderlabs/ssh"

	"ssh-games/internal/lobby"
	"ssh-games/internal/sshutil"
)

type Server struct {
	manager *lobby.Manager
	hostKey gossh.Option
}

func NewServer(manager *lobby.Manager) (*Server, error) {
	hostKey, err := sshutil.EnsureHostKey("/data/hub_host_key")
	if err != nil {
		return nil, fmt.Errorf("host key: %w", err)
	}

	return &Server{manager: manager, hostKey: hostKey}, nil
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &gossh.Server{
		Addr:    addr,
		Handler: s.handleSession,
		PasswordHandler: func(gossh.Context, string) bool {
			return true
		},
		PublicKeyHandler: func(gossh.Context, gossh.PublicKey) bool {
			return true
		},
		PtyCallback: func(gossh.Context, gossh.Pty) bool {
			return true
		},
	}
	if err := srv.SetOption(s.hostKey); err != nil {
		return fmt.Errorf("set host key: %w", err)
	}

	log.Printf("Hub SSH listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) handleSession(sess gossh.Session) {
	pty, winCh, hasPTY := sess.Pty()
	if !hasPTY {
		io.WriteString(sess, "PTY required\r\n")
		return
	}

	nick := sess.User()
	if nick == "" {
		nick = "player"
	}
	nick = sanitizeNick(nick)

	m := newMenu(sess, s.manager, nick, pty, winCh)
	m.run()
}

func sanitizeNick(nick string) string {
	if len(nick) > 12 {
		nick = nick[:12]
	}
	runes := []rune(nick)
	result := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r >= 32 && r != '[' && r != ']' {
			result = append(result, r)
		}
	}
	if len(result) == 0 {
		return "player"
	}
	return string(result)
}
