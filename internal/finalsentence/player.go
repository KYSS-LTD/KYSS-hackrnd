package finalsentence

import "io"

type Player struct {
	Nick   string
	output io.Writer
	send   chan []byte
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
	close(p.send)
}
