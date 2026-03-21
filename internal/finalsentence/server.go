package finalsentence

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"unicode/utf8"

	gossh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	game    *Game
	hostKey ssh.Signer
}

func NewServer() (*Server, error) {
	game := NewGame()

	key, err := generateEphemeralKey()
	if err != nil {
		return nil, fmt.Errorf("key: %w", err)
	}

	return &Server{game: game, hostKey: key}, nil
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
	srv.AddHostKey(s.hostKey)
	log.Printf("Final sentence SSH listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) handleSession(sess gossh.Session) {
	_, _, hasPTY := sess.Pty()
	if !hasPTY {
		io.WriteString(sess, "PTY required\r\n")
		return
	}

	nick := sanitizeNick(sess.User())
	player := newPlayer(nick, sess)
	s.game.Join(nick, player)
	defer s.game.Leave(nick)

	io.WriteString(sess, "\x1b[?1049h")
	io.WriteString(sess, "\x1b[?25l")
	io.WriteString(sess, "\x1b[2J\x1b[H")
	defer func() {
		io.WriteString(sess, "\x1b[?25h")
		io.WriteString(sess, "\x1b[?1049l")
	}()

	buf := make([]byte, 8)
	for {
		n, err := sess.Read(buf)
		if err != nil {
			return
		}
		key := parseInput(buf[:n])
		if key == "ctrl-c" || key == "q" || key == "Q" {
			return
		}
		s.game.Input(nick, key)
	}
}

func parseInput(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) >= 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'A':
			return "up"
		case 'B':
			return "down"
		case 'C':
			return "right"
		case 'D':
			return "left"
		}
	}
	r, _ := utf8.DecodeRune(b)
	switch r {
	case 3:
		return "ctrl-c"
	case 13, 10:
		return "enter"
	case 127, 8:
		return "backspace"
	default:
		return string(r)
	}
}

func sanitizeNick(nick string) string {
	if nick == "" {
		nick = "player"
	}
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

func generateEphemeralKey() (ssh.Signer, error) {
	keyFile := "/data/finalsentence_host_key"
	data, err := os.ReadFile(keyFile)
	if err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err == nil {
				return ssh.NewSignerFromKey(key)
			}
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if err := os.MkdirAll("/data", 0700); err == nil {
		os.WriteFile(keyFile, privBytes, 0600)
	}

	return ssh.NewSignerFromKey(privateKey)
}
