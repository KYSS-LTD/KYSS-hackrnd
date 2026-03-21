package snake

import (
	"fmt"
	"sync"
	"time"
)

const (
	tickRate        = 150 * time.Millisecond
	maxSnakePlayers = 6
)

type Game struct {
	mu           sync.RWMutex
	state        *GameState
	players      map[string]*Player
	nickOrder    []string
	deadPlayers  map[string]bool
	playerColors map[string]int
	freeColors   []int
	inputCh      chan inputEvent
	joinCh       chan joinEvent
	leaveCh      chan string
}

type inputEvent struct {
	nick string
	key  string
}

type joinEvent struct {
	nick   string
	player *Player
	reply  chan error
}

func NewGame() *Game {
	g := &Game{
		state:        newGameState(),
		players:      make(map[string]*Player),
		deadPlayers:  make(map[string]bool),
		playerColors: make(map[string]int),
		freeColors:   []int{0, 1, 2, 3, 4, 5},
		inputCh:      make(chan inputEvent, 64),
		joinCh:       make(chan joinEvent, 8),
		leaveCh:      make(chan string, 8),
	}
	go g.loop()
	return g
}

func (g *Game) Join(nick string, p *Player) error {
	reply := make(chan error, 1)
	g.joinCh <- joinEvent{nick: nick, player: p, reply: reply}
	return <-reply
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
			err := g.handleJoin(ev)
			g.mu.Unlock()
			ev.reply <- err

		case nick := <-g.leaveCh:
			g.mu.Lock()
			if p, ok := g.players[nick]; ok {
				p.close()
			}
			delete(g.players, nick)
			g.state.removePlayer(nick)
			delete(g.deadPlayers, nick)
			if colorIdx, ok := g.playerColors[nick]; ok {
				delete(g.playerColors, nick)
				g.releaseColor(colorIdx)
			}
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

func (g *Game) handleJoin(ev joinEvent) error {
	if _, exists := g.players[ev.nick]; exists {
		return fmt.Errorf("nick %q is already online", ev.nick)
	}
	if len(g.players) >= maxSnakePlayers {
		return fmt.Errorf("lobby is full")
	}
	colorIdx, ok := g.claimColor()
	if !ok {
		return fmt.Errorf("no color slots available")
	}

	g.players[ev.nick] = ev.player
	g.playerColors[ev.nick] = colorIdx
	g.state.addPlayer(ev.nick)
	g.nickOrder = append(g.nickOrder, ev.nick)
	g.deadPlayers[ev.nick] = false
	return nil
}

func (g *Game) handleInput(nick, key string) {
	s, ok := g.state.Snakes[nick]
	if !ok {
		return
	}

	if g.deadPlayers[nick] {
		if key == "c" || key == "C" || key == "с" || key == "С" {
			g.respawn(nick)
		}
		return
	}

	switch key {
	case "up", "w", "W", "ц", "Ц":
		if !s.Dir.Equal(DirDown) {
			s.NextDir = DirUp
		}
	case "down", "s", "S", "ы", "Ы":
		if !s.Dir.Equal(DirUp) {
			s.NextDir = DirDown
		}
	case "left", "a", "A", "ф", "Ф":
		if !s.Dir.Equal(DirRight) {
			s.NextDir = DirLeft
		}
	case "right", "d", "D", "в", "В":
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
	g.state.Scores[nick] = 0
	g.deadPlayers[nick] = false
}

func (g *Game) doTick() {
	if len(g.state.Snakes) == 0 {
		return
	}

	dead := g.state.tick()
	for nick := range dead {
		g.deadPlayers[nick] = true
		g.state.Scores[nick] = 0
	}

	stateCopy := g.state.clone()
	deadCopy := make(map[string]bool)
	for k, v := range g.deadPlayers {
		deadCopy[k] = v
	}
	orderCopy := make([]string, len(g.nickOrder))
	copy(orderCopy, g.nickOrder)
	colorsCopy := make(map[string]int, len(g.playerColors))
	for nick, idx := range g.playerColors {
		colorsCopy[nick] = idx
	}

	for nick, player := range g.players {
		overlayNick := ""
		if deadCopy[nick] {
			overlayNick = nick
		}
		player.enqueue(renderFrame(stateCopy, deadCopy, orderCopy, colorsCopy, nick, overlayNick))
	}
}

func (g *Game) claimColor() (int, bool) {
	if len(g.freeColors) == 0 {
		return 0, false
	}
	idx := g.freeColors[0]
	g.freeColors = g.freeColors[1:]
	return idx, true
}

func (g *Game) releaseColor(idx int) {
	for _, existing := range g.freeColors {
		if existing == idx {
			return
		}
	}
	g.freeColors = append(g.freeColors, idx)
	for i := 1; i < len(g.freeColors); i++ {
		j := i
		for j > 0 && g.freeColors[j-1] > g.freeColors[j] {
			g.freeColors[j-1], g.freeColors[j] = g.freeColors[j], g.freeColors[j-1]
			j--
		}
	}
}
