package tanchiki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"unicode/utf8"

	gossh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	game    *Game
	hostKey ssh.Signer
}

func NewServer() (*Server, error) {
	key, err := generateEphemeralKey()
	if err != nil {
		return nil, fmt.Errorf("key: %w", err)
	}
	return &Server{game: NewGame(), hostKey: key}, nil
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &gossh.Server{
		Addr:             addr,
		Handler:          s.handleSession,
		PasswordHandler:  func(ctx gossh.Context, password string) bool { return true },
		PublicKeyHandler: func(ctx gossh.Context, key gossh.PublicKey) bool { return true },
		PtyCallback:      func(ctx gossh.Context, pty gossh.Pty) bool { return true },
	}
	srv.AddHostKey(s.hostKey)
	log.Printf("Tanchiki SSH listening on %s", addr)
	return srv.ListenAndServe()
}

func (s *Server) handleSession(sess gossh.Session) {
	_, _, ok := sess.Pty()
	if !ok {
		_, _ = io.WriteString(sess, "PTY required\r\n")
		return
	}
	nick := sanitizeNick(sess.User())
	player := newPlayer(nick, sess)
	if err := s.game.Join(nick, player); err != nil {
		player.close()
		msg := "\r\nне удалось подключиться\r\n"
		if err.Error() == "lobby is full" {
			msg = "\r\nошибка: лобби заполнено\r\n"
		}
		_, _ = io.WriteString(sess, msg)
		_, _ = io.WriteString(sess, "соединение будет закрыто через 3 секунды...\r\n")
		time.Sleep(3 * time.Second)
		return
	}
	defer s.game.Leave(nick)

	_, _ = io.WriteString(sess, "\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
	defer io.WriteString(sess, "\x1b[?25h\x1b[?1049l")

	buf := make([]byte, 8)
	for {
		n, err := sess.Read(buf)
		if err != nil {
			return
		}
		key := parseInput(buf[:n])
		if key == "ctrl-c" || key == "q" || key == "Q" || key == "й" || key == "Й" {
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
	default:
		return string(r)
	}
}

func sanitizeNick(nick string) string {
	if len(nick) > 12 {
		nick = nick[:12]
	}
	if nick == "" {
		return "player"
	}
	return nick
}

func generateEphemeralKey() (ssh.Signer, error) {
	keyFile := "/data/tanchiki_host_key"
	if data, err := os.ReadFile(keyFile); err == nil {
		if block, _ := pem.Decode(data); block != nil {
			if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
				return ssh.NewSignerFromKey(key)
			}
		}
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	privBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.MkdirAll("/data", 0700); err == nil {
		_ = os.WriteFile(keyFile, privBytes, 0600)
	}
	return ssh.NewSignerFromKey(privateKey)
}
