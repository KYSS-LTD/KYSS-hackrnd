package hub

import (
	"fmt"
	"io"
	"log"
	"net"

	"ssh-games/internal/lobby"

	gossh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
)

type proxy struct {
	manager *lobby.Manager
	lobbyID string
}

func newProxy(m *lobby.Manager, lobbyID string) *proxy {
	return &proxy{manager: m, lobbyID: lobbyID}
}

func (p *proxy) connect(sess gossh.Session, nick string, pty gossh.Pty, winCh <-chan gossh.Window) {
	lob, ok := p.manager.GetLobby(p.lobbyID)
	if !ok {
		io.WriteString(sess, "\r\nлобби не найдено\r\n")
		return
	}

	conn, err := net.Dial("tcp", lob.Addr)
	if err != nil {
		io.WriteString(sess, fmt.Sprintf("\r\nне удалось подключиться: %v\r\n", err))
		return
	}
	defer conn.Close()

	config := &ssh.ClientConfig{
		User:            nick,
		Auth:            []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, lob.Addr, config)
	if err != nil {
		io.WriteString(sess, fmt.Sprintf("\r\nSSH handshake failed: %v\r\n", err))
		return
	}
	defer clientConn.Close()

	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()

	gameSession, err := client.NewSession()
	if err != nil {
		io.WriteString(sess, fmt.Sprintf("\r\nне удалось открыть сессию: %v\r\n", err))
		return
	}
	defer gameSession.Close()

	stdin, err := gameSession.StdinPipe()
	if err != nil {
		log.Printf("stdin pipe: %v", err)
		return
	}
	stdout, err := gameSession.StdoutPipe()
	if err != nil {
		log.Printf("stdout pipe: %v", err)
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	termEnv := pty.Term
	if termEnv == "" {
		termEnv = "xterm-256color"
	}
	if err := gameSession.RequestPty(termEnv, pty.Window.Height, pty.Window.Width, modes); err != nil {
		io.WriteString(sess, fmt.Sprintf("\r\nPTY failed: %v\r\n", err))
		return
	}

	if err := gameSession.Shell(); err != nil {
		io.WriteString(sess, fmt.Sprintf("\r\nshell failed: %v\r\n", err))
		return
	}

	p.manager.IncrementPlayers(p.lobbyID, 1)
	defer p.manager.IncrementPlayers(p.lobbyID, -1)

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(stdin, sess)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(sess, stdout)
		done <- struct{}{}
	}()

	go func() {
		for w := range winCh {
			gameSession.WindowChange(w.Height, w.Width)
		}
	}()

	<-done
	gameSession.Close()
}
