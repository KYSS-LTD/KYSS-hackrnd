package snake

import (
	"io"
	"sync"
)

type Player struct {
	Nick      string
	output    io.Writer
	send      chan []byte
	dead      bool
	mu        sync.Mutex
	closeOnce sync.Once
}

func newPlayer(nick string, output io.Writer) *Player {
	p := &Player{
		Nick:   nick,
		output: output,
		send:   make(chan []byte, 32),
	}
	go p.writeLoop()
	return p
}

func (p *Player) writeLoop() {
	for data := range p.send {
		p.output.Write(data)
	}
}

func (p *Player) enqueue(data []byte) {
	select {
	case p.send <- data:
	default:
	}
}

func (p *Player) close() {
	p.closeOnce.Do(func() {
		close(p.send)
	})
}

func (p *Player) isDead() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dead
}

func (p *Player) setDead(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dead = v
}
