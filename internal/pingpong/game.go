package pingpong

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	boardWidth   = 48
	boardHeight  = 16
	paddleHeight = 4
	tickRate     = 60 * time.Millisecond
	winningScore = 7
)

type side string

const (
	sideLeft     side = "left"
	sideRight    side = "right"
	sideSpectate side = "spectator"
)

type playerState struct {
	player *Player
	side   side
}

type Game struct {
	mu          sync.Mutex
	players     map[string]*playerState
	joinOrder   []string
	inputCh     chan inputEvent
	joinCh      chan joinEvent
	leaveCh     chan string
	leftPaddle  int
	rightPaddle int
	ballX       float64
	ballY       float64
	ballVX      float64
	ballVY      float64
	leftScore   int
	rightScore  int
	status      string
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
		players: make(map[string]*playerState),
		inputCh: make(chan inputEvent, 64),
		joinCh:  make(chan joinEvent, 8),
		leaveCh: make(chan string, 8),
	}
	g.resetRound(1)
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
			g.players[ev.nick] = &playerState{player: ev.player, side: g.assignSide()}
			g.joinOrder = append(g.joinOrder, ev.nick)
			g.status = g.statusLine()
			g.broadcastLocked()
			g.mu.Unlock()
		case nick := <-g.leaveCh:
			g.mu.Lock()
			if p, ok := g.players[nick]; ok {
				p.player.close()
				delete(g.players, nick)
			}
			g.compactJoinOrder()
			g.rebalanceSides()
			g.status = g.statusLine()
			g.broadcastLocked()
			g.mu.Unlock()
		case ev := <-g.inputCh:
			g.mu.Lock()
			g.handleInput(ev.nick, ev.key)
			g.mu.Unlock()
		case <-ticker.C:
			g.mu.Lock()
			g.step()
			g.broadcastLocked()
			g.mu.Unlock()
		}
	}
}

func (g *Game) assignSide() side {
	hasLeft := false
	hasRight := false
	for _, p := range g.players {
		switch p.side {
		case sideLeft:
			hasLeft = true
		case sideRight:
			hasRight = true
		}
	}
	if !hasLeft {
		return sideLeft
	}
	if !hasRight {
		return sideRight
	}
	return sideSpectate
}

func (g *Game) compactJoinOrder() {
	filtered := g.joinOrder[:0]
	for _, nick := range g.joinOrder {
		if _, ok := g.players[nick]; ok {
			filtered = append(filtered, nick)
		}
	}
	g.joinOrder = filtered
}

func (g *Game) rebalanceSides() {
	for _, p := range g.players {
		p.side = sideSpectate
	}
	for _, nick := range g.joinOrder {
		p := g.players[nick]
		if p == nil {
			continue
		}
		if p.side == sideSpectate {
			p.side = g.assignSide()
		}
	}
}

func (g *Game) handleInput(nick, key string) {
	p := g.players[nick]
	if p == nil {
		return
	}

	switch key {
	case "r", "R":
		if g.leftScore >= winningScore || g.rightScore >= winningScore {
			g.leftScore = 0
			g.rightScore = 0
			g.resetRound(1)
			g.status = "новый матч начался"
		}
		return
	}

	delta := 0
	switch key {
	case "up", "w", "W", "k", "K":
		delta = -1
	case "down", "s", "S", "j", "J":
		delta = 1
	default:
		return
	}

	switch p.side {
	case sideLeft:
		g.leftPaddle = clamp(g.leftPaddle+delta, 1, boardHeight-paddleHeight+1)
	case sideRight:
		g.rightPaddle = clamp(g.rightPaddle+delta, 1, boardHeight-paddleHeight+1)
	}
}

func (g *Game) step() {
	if !g.hasPlayer(sideRight) {
		g.trackBallWithAI()
	}
	if !g.hasPlayer(sideLeft) {
		g.leftPaddle = centerPaddle()
	}
	if g.leftScore >= winningScore || g.rightScore >= winningScore {
		g.status = fmt.Sprintf("матч окончен — нажмите R для рестарта")
		return
	}

	g.ballX += g.ballVX
	g.ballY += g.ballVY

	if g.ballY <= 1 {
		g.ballY = 1
		g.ballVY *= -1
	}
	if g.ballY >= boardHeight {
		g.ballY = boardHeight
		g.ballVY *= -1
	}

	if g.ballVX < 0 && g.ballX <= 2 {
		if g.ballY >= float64(g.leftPaddle) && g.ballY <= float64(g.leftPaddle+paddleHeight-1) {
			g.ballX = 2
			g.ballVX = minAbs(g.ballVX*-1.05, 1.6)
			g.ballVY = clampFloat(g.ballVY+(g.ballY-float64(g.leftPaddle)-1.5)*0.05, -0.75, 0.75)
		} else {
			g.rightScore++
			g.resetRound(-1)
			g.status = g.statusLine()
			return
		}
	}

	if g.ballVX > 0 && g.ballX >= boardWidth-1 {
		if g.ballY >= float64(g.rightPaddle) && g.ballY <= float64(g.rightPaddle+paddleHeight-1) {
			g.ballX = boardWidth - 1
			g.ballVX = -minAbs(g.ballVX*1.05, 1.6)
			g.ballVY = clampFloat(g.ballVY+(g.ballY-float64(g.rightPaddle)-1.5)*0.05, -0.75, 0.75)
		} else {
			g.leftScore++
			g.resetRound(1)
			g.status = g.statusLine()
			return
		}
	}

	g.status = g.statusLine()
}

func (g *Game) trackBallWithAI() {
	center := float64(g.rightPaddle) + float64(paddleHeight-1)/2
	if g.ballY < center-0.35 {
		g.rightPaddle = clamp(g.rightPaddle-1, 1, boardHeight-paddleHeight+1)
	} else if g.ballY > center+0.35 {
		g.rightPaddle = clamp(g.rightPaddle+1, 1, boardHeight-paddleHeight+1)
	}
}

func (g *Game) resetRound(direction float64) {
	g.leftPaddle = centerPaddle()
	g.rightPaddle = centerPaddle()
	g.ballX = boardWidth / 2
	g.ballY = boardHeight / 2
	g.ballVX = 0.8 * direction
	g.ballVY = 0.35
}

func (g *Game) hasPlayer(target side) bool {
	for _, p := range g.players {
		if p.side == target {
			return true
		}
	}
	return false
}

func (g *Game) statusLine() string {
	left := g.playerName(sideLeft, "ждём игрока")
	rightFallback := "бот"
	if g.hasPlayer(sideRight) {
		rightFallback = "игрок"
	}
	right := g.playerName(sideRight, rightFallback)
	if g.leftScore >= winningScore {
		return fmt.Sprintf("%s победил", left)
	}
	if g.rightScore >= winningScore {
		return fmt.Sprintf("%s победил", right)
	}
	if !g.hasPlayer(sideRight) {
		return "режим тренировки против бота"
	}
	return fmt.Sprintf("%s vs %s", left, right)
}

func (g *Game) playerName(target side, fallback string) string {
	for _, nick := range g.joinOrder {
		if p := g.players[nick]; p != nil && p.side == target {
			return nick
		}
	}
	return fallback
}

func (g *Game) broadcastLocked() {
	order := append([]string(nil), g.joinOrder...)
	sort.Strings(order)
	for nick, p := range g.players {
		p.player.enqueue(renderFrame(frameState{
			nick:        nick,
			side:        p.side,
			leftPaddle:  g.leftPaddle,
			rightPaddle: g.rightPaddle,
			ballX:       int(g.ballX + 0.5),
			ballY:       int(g.ballY + 0.5),
			leftScore:   g.leftScore,
			rightScore:  g.rightScore,
			status:      g.status,
			players:     order,
			leftName:    g.playerName(sideLeft, "open"),
			rightName:   g.playerName(sideRight, "bot"),
		}))
	}
}

func centerPaddle() int { return boardHeight/2 - paddleHeight/2 + 1 }

func clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func minAbs(v, maxAbs float64) float64 {
	if v > maxAbs {
		return maxAbs
	}
	if v < -maxAbs {
		return -maxAbs
	}
	return v
}
