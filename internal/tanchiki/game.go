package tanchiki

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

const (
	arenaW           = 42
	arenaH           = 20
	maxTanchikiUsers = 6
	tickRate         = 180 * time.Millisecond
	respawnTicks     = 18
	fireCooldown     = 4
)

type dir struct{ dx, dy int }

var (
	dirUp    = dir{0, -1}
	dirDown  = dir{0, 1}
	dirLeft  = dir{-1, 0}
	dirRight = dir{1, 0}
)

type point struct{ x, y int }

type tank struct {
	nick      string
	pos       point
	dir       dir
	alive     bool
	respawnIn int
	cooldown  int
	score     int
	deaths    int
	colorIdx  int
}

type bullet struct {
	pos   point
	dir   dir
	owner string
}

type inputState struct {
	move    dir
	moveTTL int
	fire    bool
}

type Game struct {
	mu sync.Mutex

	players map[string]*Player
	tanks   map[string]*tank
	order   []string

	walls   map[point]int
	barrels map[point]bool
	bullets []bullet

	joinCh  chan joinEvent
	leaveCh chan string
	inputCh chan inputEvent

	rng *rand.Rand
}

type joinEvent struct {
	nick   string
	player *Player
	reply  chan error
}

type inputEvent struct {
	nick string
	key  string
}

func NewGame() *Game {
	g := &Game{
		players: make(map[string]*Player),
		tanks:   make(map[string]*tank),
		walls:   make(map[point]int),
		barrels: make(map[point]bool),
		joinCh:  make(chan joinEvent, 8),
		leaveCh: make(chan string, 8),
		inputCh: make(chan inputEvent, 64),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	g.generateArena()
	go g.loop()
	return g
}

func (g *Game) Join(nick string, p *Player) error {
	reply := make(chan error, 1)
	g.joinCh <- joinEvent{nick: nick, player: p, reply: reply}
	return <-reply
}

func (g *Game) Leave(nick string) { g.leaveCh <- nick }

func (g *Game) Input(nick, key string) {
	select {
	case g.inputCh <- inputEvent{nick: nick, key: key}:
	default:
	}
}

func (g *Game) loop() {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	inputs := make(map[string]inputState)

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
				delete(g.players, nick)
			}
			delete(g.tanks, nick)
			delete(inputs, nick)
			g.order = removeNick(g.order, nick)
			g.mu.Unlock()
		case ev := <-g.inputCh:
			g.mu.Lock()
			in := inputs[ev.nick]
			switch ev.key {
			case "up", "w", "W", "ц", "Ц":
				in.move = dirUp
				in.moveTTL = 1
			case "down", "s", "S", "ы", "Ы":
				in.move = dirDown
				in.moveTTL = 1
			case "left", "a", "A", "ф", "Ф":
				in.move = dirLeft
				in.moveTTL = 1
			case "right", "d", "D", "в", "В":
				in.move = dirRight
				in.moveTTL = 1
			case " ", "f", "F", "а", "А":
				in.fire = true
			}
			inputs[ev.nick] = in
			g.mu.Unlock()
		case <-ticker.C:
			g.mu.Lock()
			g.tick(inputs)
			for nick := range inputs {
				st := inputs[nick]
				if st.moveTTL > 0 {
					st.moveTTL--
					if st.moveTTL == 0 {
						st.move = dir{}
					}
				}
				st.fire = false
				inputs[nick] = st
			}
			g.broadcastLocked()
			g.mu.Unlock()
		}
	}
}

func (g *Game) handleJoin(ev joinEvent) error {
	if _, exists := g.players[ev.nick]; exists {
		return fmt.Errorf("nick %q is already online", ev.nick)
	}
	if len(g.players) >= maxTanchikiUsers {
		return fmt.Errorf("lobby is full")
	}
	g.players[ev.nick] = ev.player
	g.order = append(g.order, ev.nick)
	spawn := g.findSpawn()
	g.tanks[ev.nick] = &tank{nick: ev.nick, pos: spawn, dir: dirUp, alive: true, colorIdx: (len(g.order) - 1) % len(tankColors)}
	return nil
}

func (g *Game) tick(inputs map[string]inputState) {
	g.stepBullets()

	for nick, t := range g.tanks {
		if !t.alive {
			t.respawnIn--
			if t.respawnIn <= 0 {
				t.pos = g.findSpawn()
				t.alive = true
				t.cooldown = fireCooldown
				t.dir = dirUp
			}
			continue
		}
		if t.cooldown > 0 {
			t.cooldown--
		}
		in := inputs[nick]
		if in.move != (dir{}) {
			t.dir = in.move
			np := point{t.pos.x + in.move.dx, t.pos.y + in.move.dy}
			if g.canMoveTo(np, nick) {
				t.pos = np
			}
		}
		if in.fire && t.cooldown == 0 {
			bp := point{t.pos.x + t.dir.dx, t.pos.y + t.dir.dy}
			if g.inBounds(bp) {
				g.spawnBullet(bp, t.dir, nick)
			}
		}
	}
}

func (g *Game) spawnBullet(pos point, d dir, owner string) {
	if hp, ok := g.walls[pos]; ok {
		hp--
		if hp <= 0 {
			delete(g.walls, pos)
		} else {
			g.walls[pos] = hp
		}
	} else if g.barrels[pos] {
		g.explode(pos, owner)
	} else {
		for nick, t := range g.tanks {
			if t.alive && t.pos == pos {
				g.killTank(nick, owner)
				if shooter := g.tanks[owner]; shooter != nil {
					shooter.cooldown = fireCooldown
				}
				return
			}
		}
		g.bullets = append(g.bullets, bullet{pos: pos, dir: d, owner: owner})
	}
	if shooter := g.tanks[owner]; shooter != nil {
		shooter.cooldown = fireCooldown
	}
}

func (g *Game) stepBullets() {
	next := make([]bullet, 0, len(g.bullets))
	for _, b := range g.bullets {
		np := point{b.pos.x + b.dir.dx, b.pos.y + b.dir.dy}
		if !g.inBounds(np) {
			continue
		}
		if hp, ok := g.walls[np]; ok {
			hp--
			if hp <= 0 {
				delete(g.walls, np)
			} else {
				g.walls[np] = hp
			}
			continue
		}
		if g.barrels[np] {
			g.explode(np, b.owner)
			continue
		}
		hit := false
		for nick, t := range g.tanks {
			if t.alive && t.pos == np {
				g.killTank(nick, b.owner)
				hit = true
				break
			}
		}
		if hit {
			continue
		}
		b.pos = np
		next = append(next, b)
	}
	g.bullets = next
}

func (g *Game) explode(center point, owner string) {
	queue := []point{center}
	seen := map[point]bool{}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if seen[c] {
			continue
		}
		seen[c] = true
		delete(g.barrels, c)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				p := point{c.x + dx, c.y + dy}
				if !g.inBounds(p) {
					continue
				}
				if hp, ok := g.walls[p]; ok {
					hp--
					if hp <= 0 {
						delete(g.walls, p)
					} else {
						g.walls[p] = hp
					}
				}
				if g.barrels[p] && !seen[p] {
					queue = append(queue, p)
				}
				for nick, t := range g.tanks {
					if t.alive && t.pos == p {
						g.killTank(nick, owner)
					}
				}
			}
		}
	}
}

func (g *Game) killTank(victim, killer string) {
	t := g.tanks[victim]
	if t == nil || !t.alive {
		return
	}
	t.alive = false
	t.respawnIn = respawnTicks
	t.deaths++
	if killerTank := g.tanks[killer]; killer != "" && killer != victim && killerTank != nil {
		killerTank.score++
	}
}

func (g *Game) canMoveTo(p point, nick string) bool {
	if !g.inBounds(p) || g.walls[p] > 0 || g.barrels[p] {
		return false
	}
	for other, t := range g.tanks {
		if other != nick && t.alive && t.pos == p {
			return false
		}
	}
	return true
}

func (g *Game) inBounds(p point) bool {
	return p.x >= 1 && p.x <= arenaW && p.y >= 1 && p.y <= arenaH
}

func (g *Game) findSpawn() point {
	for i := 0; i < 120; i++ {
		p := point{x: 1 + g.rng.Intn(arenaW), y: 1 + g.rng.Intn(arenaH)}
		if g.walls[p] == 0 && !g.barrels[p] {
			occupied := false
			for _, t := range g.tanks {
				if t.alive && t.pos == p {
					occupied = true
					break
				}
			}
			if !occupied {
				return p
			}
		}
	}
	return point{2, 2}
}

func (g *Game) generateArena() {
	for y := 2; y < arenaH; y += 2 {
		for x := 3; x < arenaW; x += 4 {
			if g.rng.Intn(100) < 62 {
				g.walls[point{x, y}] = 2
				if x+1 <= arenaW && g.rng.Intn(100) < 45 {
					g.walls[point{x + 1, y}] = 2
				}
			}
		}
	}
	for i := 0; i < 18; i++ {
		p := point{x: 2 + g.rng.Intn(arenaW-2), y: 2 + g.rng.Intn(arenaH-2)}
		if g.walls[p] == 0 {
			g.barrels[p] = true
		}
	}
}

func (g *Game) broadcastLocked() {
	order := append([]string(nil), g.order...)
	sort.Strings(order)
	for nick, p := range g.players {
		p.enqueue(renderFrame(g, nick, order))
	}
}

func removeNick(list []string, nick string) []string {
	out := list[:0]
	for _, v := range list {
		if v != nick {
			out = append(out, v)
		}
	}
	return out
}
