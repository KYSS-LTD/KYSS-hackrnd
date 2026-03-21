package finalsentence

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	roundDuration = 60 * time.Second
	tickRate      = 100 * time.Millisecond
	inputLockTime = 3 * time.Second
)

var roundTexts = [][]string{
	{"final sentence", "green beats red", "speed wins rounds", "focus on rhythm", "type the future"},
	{"signal stays clear", "letters chase time", "tiny words matter", "rivals push harder", "finish every line"},
	{"quiet hands move", "perfect timing helps", "white text turns green", "mistakes glow red", "one more sentence"},
}

type inputEvent struct {
	nick string
	key  string
}

type joinEvent struct {
	nick   string
	player *Player
}

type typedRune struct {
	value   rune
	correct bool
}

type mistakeState struct {
	value rune
	index int
}

type playerProgress struct {
	lineIndex    int
	typed        []typedRune
	mistake      *mistakeState
	blockedUntil time.Time
	finished     bool
	finishedAt   time.Time
}

type Game struct {
	mu          sync.Mutex
	players     map[string]*Player
	progress    map[string]*playerProgress
	joinOrder   []string
	inputCh     chan inputEvent
	joinCh      chan joinEvent
	leaveCh     chan string
	roundIndex  int
	roundStart  time.Time
	lastWinners []string
}

func NewGame() *Game {
	g := &Game{
		players:  make(map[string]*Player),
		progress: make(map[string]*playerProgress),
		inputCh:  make(chan inputEvent, 128),
		joinCh:   make(chan joinEvent, 16),
		leaveCh:  make(chan string, 16),
	}
	g.roundIndex = -1
	g.startRound(time.Now())
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
			if old := g.players[ev.nick]; old != nil {
				old.close()
			}
			g.players[ev.nick] = ev.player
			g.progress[ev.nick] = &playerProgress{}
			g.joinOrder = appendUniqueNick(g.joinOrder, ev.nick)
			g.broadcastLocked(time.Now())
			g.mu.Unlock()
		case nick := <-g.leaveCh:
			g.mu.Lock()
			if p := g.players[nick]; p != nil {
				p.close()
				delete(g.players, nick)
			}
			delete(g.progress, nick)
			g.joinOrder = removeNick(g.joinOrder, nick)
			g.broadcastLocked(time.Now())
			g.mu.Unlock()
		case ev := <-g.inputCh:
			g.mu.Lock()
			g.handleInput(ev.nick, ev.key, time.Now())
			g.broadcastLocked(time.Now())
			g.mu.Unlock()
		case now := <-ticker.C:
			g.mu.Lock()
			g.clearExpiredMistakesLocked(now)
			if now.Sub(g.roundStart) >= roundDuration {
				g.finishRoundLocked(now)
				g.startRound(now)
			}
			g.broadcastLocked(now)
			g.mu.Unlock()
		}
	}
}

func (g *Game) handleInput(nick, key string, now time.Time) {
	prog := g.progress[nick]
	if prog == nil || prog.finished || now.Sub(g.roundStart) >= roundDuration {
		return
	}
	if key == "enter" || key == "up" || key == "down" || key == "left" || key == "right" {
		return
	}
	if !prog.blockedUntil.IsZero() && !now.Before(prog.blockedUntil) {
		prog.mistake = nil
		prog.blockedUntil = time.Time{}
	}
	if now.Before(prog.blockedUntil) {
		return
	}
	if key == "backspace" {
		if len(prog.typed) > 0 {
			prog.typed = prog.typed[:len(prog.typed)-1]
		}
		prog.mistake = nil
		return
	}

	target := []rune(g.currentLines()[prog.lineIndex])
	if len(prog.typed) >= len(target) {
		return
	}
	r := []rune(key)
	if len(r) != 1 || r[0] < 32 || r[0] > 126 {
		return
	}

	idx := len(prog.typed)
	if target[idx] != r[0] {
		prog.mistake = &mistakeState{value: r[0], index: idx}
		prog.blockedUntil = now.Add(inputLockTime)
		return
	}

	prog.typed = append(prog.typed, typedRune{value: r[0], correct: true})
	prog.mistake = nil
	prog.blockedUntil = time.Time{}
	if len(prog.typed) == len(target) {
		if prog.lineIndex == len(g.currentLines())-1 {
			prog.finished = true
			prog.finishedAt = now
		} else {
			prog.lineIndex++
			prog.typed = nil
			prog.mistake = nil
			prog.blockedUntil = time.Time{}
		}
	}
}

func (g *Game) clearExpiredMistakesLocked(now time.Time) {
	for _, prog := range g.progress {
		if prog == nil || prog.blockedUntil.IsZero() || now.Before(prog.blockedUntil) {
			continue
		}
		prog.mistake = nil
		prog.blockedUntil = time.Time{}
	}
}

func (g *Game) finishRoundLocked(now time.Time) {
	type score struct {
		nick       string
		completion float64
		finished   bool
		finishedAt time.Time
	}
	var scores []score
	for nick, prog := range g.progress {
		scores = append(scores, score{
			nick:       nick,
			completion: progressRatioForLines(g.currentLines(), prog),
			finished:   prog.finished,
			finishedAt: prog.finishedAt,
		})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].finished != scores[j].finished {
			return scores[i].finished
		}
		if scores[i].finished && scores[j].finished && !scores[i].finishedAt.Equal(scores[j].finishedAt) {
			return scores[i].finishedAt.Before(scores[j].finishedAt)
		}
		if scores[i].completion != scores[j].completion {
			return scores[i].completion > scores[j].completion
		}
		return scores[i].nick < scores[j].nick
	})
	g.lastWinners = g.lastWinners[:0]
	if len(scores) == 0 {
		return
	}
	best := scores[0].completion
	bestFinished := scores[0].finished
	bestTime := scores[0].finishedAt
	for _, s := range scores {
		if s.finished != bestFinished {
			break
		}
		if s.finished {
			if !s.finishedAt.Equal(bestTime) {
				break
			}
		} else if s.completion != best {
			break
		}
		g.lastWinners = append(g.lastWinners, s.nick)
	}
	_ = now
}

func (g *Game) startRound(now time.Time) {
	g.roundStart = now
	g.roundIndex = (g.roundIndex + 1) % len(roundTexts)
	for nick := range g.players {
		g.progress[nick] = &playerProgress{}
	}
}

func (g *Game) currentLines() []string {
	return roundTexts[g.roundIndex]
}

func (g *Game) broadcastLocked(now time.Time) {
	state := g.snapshotLocked(now)
	for nick, p := range g.players {
		p.enqueue(renderFrame(state, nick))
	}
}

func (g *Game) snapshotLocked(now time.Time) roundState {
	lines := append([]string(nil), g.currentLines()...)
	players := make([]playerSnapshot, 0, len(g.players))
	leaderRatio := 0.0
	for _, nick := range g.joinOrder {
		prog := g.progress[nick]
		if prog == nil {
			continue
		}
		ratio := progressRatioForLines(lines, prog)
		if ratio > leaderRatio {
			leaderRatio = ratio
		}
		players = append(players, playerSnapshot{
			Nick:         nick,
			LineIndex:    prog.lineIndex,
			Typed:        append([]typedRune(nil), prog.typed...),
			Mistake:      prog.mistake,
			InputBlocked: now.Before(prog.blockedUntil),
			Finished:     prog.finished,
			FinishedAt:   prog.finishedAt,
			Ratio:        ratio,
		})
	}
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Finished != players[j].Finished {
			return players[i].Finished
		}
		if players[i].Finished && players[j].Finished && !players[i].FinishedAt.Equal(players[j].FinishedAt) {
			return players[i].FinishedAt.Before(players[j].FinishedAt)
		}
		if players[i].Ratio != players[j].Ratio {
			return players[i].Ratio > players[j].Ratio
		}
		return players[i].Nick < players[j].Nick
	})

	remaining := roundDuration - now.Sub(g.roundStart)
	if remaining < 0 {
		remaining = 0
	}
	return roundState{
		Lines:       lines,
		RoundNumber: g.roundIndex + 1,
		Remaining:   remaining,
		Duration:    roundDuration,
		Players:     players,
		LeaderRatio: leaderRatio,
		Winners:     append([]string(nil), g.lastWinners...),
	}
}

func progressRatioForLines(lines []string, prog *playerProgress) float64 {
	if prog == nil || len(lines) == 0 {
		return 0
	}
	total := 0
	done := 0
	for i, line := range lines {
		length := len([]rune(line))
		total += length
		switch {
		case i < prog.lineIndex:
			done += length
		case i == prog.lineIndex:
			done += len(prog.typed)
		}
	}
	if total == 0 {
		return 0
	}
	if done > total {
		done = total
	}
	return float64(done) / float64(total)
}

func appendUniqueNick(order []string, nick string) []string {
	for _, existing := range order {
		if existing == nick {
			return order
		}
	}
	return append(order, nick)
}

func removeNick(order []string, nick string) []string {
	filtered := order[:0]
	for _, existing := range order {
		if existing != nick {
			filtered = append(filtered, existing)
		}
	}
	return filtered
}

func winnerLabel(winners []string) string {
	if len(winners) == 0 {
		return "победитель определится в конце минуты"
	}
	return "прошлый раунд: " + strings.Join(winners, ", ")
}
