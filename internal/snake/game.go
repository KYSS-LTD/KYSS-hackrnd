package snake

import (
	"sync"
	"time"
)

const tickRate = 150 * time.Millisecond

type Game struct {
	mu          sync.RWMutex
	state       *GameState
	players     map[string]*Player
	nickOrder   []string
	deadPlayers map[string]bool
	inputCh     chan inputEvent
	joinCh      chan joinEvent
	leaveCh     chan string
}

type inputEvent struct {
	nick string
	key  string
}

type joinEvent struct {
	nick   string
	player *Player
}

func NewGame() *Game {
	g := &Game{
		state:       newGameState(),
		players:     make(map[string]*Player),
		deadPlayers: make(map[string]bool),
		inputCh:     make(chan inputEvent, 64),
		joinCh:      make(chan joinEvent, 8),
		leaveCh:     make(chan string, 8),
	}
	go g.loop()
	return g
}

func (g *Game) Join(nick string, p *Player) {
	g.joinCh <- joinEvent{nick: nick, player: p}
}

func (g *Game) Leave(nick string) {
	g.leaveCh <- nick
}

func (g *Game) Input(nick, key string) {
	select {
	case g.inputCh <- inputEvent{nick: nick, key: key}:
	default:
	}
}

func (g *Game) loop() {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for {
		select {
		case ev := <-g.joinCh:
			g.mu.Lock()
			g.players[ev.nick] = ev.player
			g.state.addPlayer(ev.nick)
			g.nickOrder = append(g.nickOrder, ev.nick)
			g.deadPlayers[ev.nick] = false
			g.mu.Unlock()

		case nick := <-g.leaveCh:
			g.mu.Lock()
			if p, ok := g.players[nick]; ok {
				p.close()
			}
			delete(g.players, nick)
			g.state.removePlayer(nick)
			delete(g.deadPlayers, nick)
			newOrder := g.nickOrder[:0]
			for _, n := range g.nickOrder {
				if n != nick {
					newOrder = append(newOrder, n)
				}
			}
			g.nickOrder = newOrder
			g.mu.Unlock()

		case ev := <-g.inputCh:
			g.mu.Lock()
			g.handleInput(ev.nick, ev.key)
			g.mu.Unlock()

		case <-ticker.C:
			g.mu.Lock()
			g.doTick()
			g.mu.Unlock()
		}
	}
}

func (g *Game) handleInput(nick, key string) {
	s, ok := g.state.Snakes[nick]
	if !ok {
		return
	}

	if g.deadPlayers[nick] {
		if key == "c" || key == "C" {
			g.respawn(nick)
		}
		return
	}

	switch key {
	case "up", "w", "W":
		if !s.Dir.Equal(DirDown) {
			s.NextDir = DirUp
		}
	case "down", "s", "S":
		if !s.Dir.Equal(DirUp) {
			s.NextDir = DirDown
		}
	case "left", "a", "A":
		if !s.Dir.Equal(DirRight) {
			s.NextDir = DirLeft
		}
	case "right", "d", "D":
		if !s.Dir.Equal(DirLeft) {
			s.NextDir = DirRight
		}
	}
}

func (g *Game) respawn(nick string) {
	pos := g.state.freeSpawnPoint()
	s := g.state.Snakes[nick]
	if s == nil {
		return
	}
	s.Body = []Point{pos, {pos.X - 1, pos.Y}}
	s.Dir = DirRight
	s.NextDir = DirRight
	s.Growing = 0
	g.deadPlayers[nick] = false
}

func (g *Game) doTick() {
	if len(g.state.Snakes) == 0 {
		return
	}

	dead := g.state.tick()
	for nick := range dead {
		g.deadPlayers[nick] = true
	}

	stateCopy := g.state.clone()
	deadCopy := make(map[string]bool)
	for k, v := range g.deadPlayers {
		deadCopy[k] = v
	}
	orderCopy := make([]string, len(g.nickOrder))
	copy(orderCopy, g.nickOrder)

	frame := renderFrame(stateCopy, deadCopy, orderCopy)

	for nick, player := range g.players {
		data := make([]byte, len(frame))
		copy(data, frame)
		if deadCopy[nick] {
			overlay := renderDeathOverlay(nick)
			data = append(data, overlay...)
		}
		player.enqueue(data)
	}
}
