package snake

import (
	"math/rand"
)

func newGameState() *GameState {
	return &GameState{
		Snakes: make(map[string]*Snake),
		Apples: []Point{},
		Scores: make(map[string]int),
	}
}

func (gs *GameState) addPlayer(nick string) {
	pos := gs.freeSpawnPoint()
	snake := &Snake{
		Body:    []Point{pos, {pos.X - 1, pos.Y}},
		Dir:     DirRight,
		NextDir: DirRight,
	}
	gs.Snakes[nick] = snake
	if _, ok := gs.Scores[nick]; !ok {
		gs.Scores[nick] = 0
	}
	gs.rebalanceApples()
}

func (gs *GameState) removePlayer(nick string) {
	delete(gs.Snakes, nick)
	gs.rebalanceApples()
}

func (gs *GameState) freeSpawnPoint() Point {
	for attempts := 0; attempts < 200; attempts++ {
		p := Point{rand.Intn(Width-4) + 2, rand.Intn(Height-2) + 1}
		if gs.isFreeAt(p) && gs.isFreeAt(Point{p.X - 1, p.Y}) {
			return p
		}
	}
	return Point{Width / 2, Height / 2}
}

func (gs *GameState) isFreeAt(p Point) bool {
	if p.X < 0 || p.X >= Width || p.Y < 0 || p.Y >= Height {
		return false
	}
	for _, s := range gs.Snakes {
		if s.Occupies(p) {
			return true
		}
	}
	for _, a := range gs.Apples {
		if a.Equal(p) {
			return false
		}
	}
	return true
}

func (gs *GameState) rebalanceApples() {
	target := len(gs.Snakes)
	if target < 1 {
		target = 1
	}
	for len(gs.Apples) < target {
		gs.spawnApple()
	}
	for len(gs.Apples) > target {
		gs.Apples = gs.Apples[:len(gs.Apples)-1]
	}
}

func (gs *GameState) spawnApple() {
	for attempts := 0; attempts < 200; attempts++ {
		p := Point{rand.Intn(Width), rand.Intn(Height)}
		free := true
		for _, s := range gs.Snakes {
			if s.Occupies(p) {
				free = false
				break
			}
		}
		if !free {
			continue
		}
		for _, a := range gs.Apples {
			if a.Equal(p) {
				free = false
				break
			}
		}
		if free {
			gs.Apples = append(gs.Apples, p)
			return
		}
	}
}

func (gs *GameState) tick() map[string]bool {
	dead := make(map[string]bool)

	newHeads := make(map[string]Point)
	for nick, s := range gs.Snakes {
		dir := s.NextDir
		if dir.Equal(opposite(s.Dir)) {
			dir = s.Dir
		}
		s.Dir = dir
		newHead := Point{s.Head().X + dir.X, s.Head().Y + dir.Y}
		newHeads[nick] = newHead
	}

	for nick, head := range newHeads {
		if head.X < 0 || head.X >= Width || head.Y < 0 || head.Y >= Height {
			dead[nick] = true
		}
	}

	for nick, head := range newHeads {
		if dead[nick] {
			continue
		}
		s := gs.Snakes[nick]
		if s.OccupiesExcludeHead(head) {
			dead[nick] = true
			continue
		}
		for otherNick, other := range gs.Snakes {
			if otherNick == nick {
				continue
			}
			if other.Occupies(head) {
				dead[nick] = true
				break
			}
		}
	}

	for nick, head := range newHeads {
		if dead[nick] {
			continue
		}
		s := gs.Snakes[nick]
		s.Body = append([]Point{head}, s.Body...)
		ateApple := false
		for i, a := range gs.Apples {
			if a.Equal(head) {
				gs.Apples = append(gs.Apples[:i], gs.Apples[i+1:]...)
				ateApple = true
				gs.Scores[nick]++
				s.Growing += 2
				break
			}
		}
		if !ateApple {
			if s.Growing > 0 {
				s.Growing--
			} else {
				s.Body = s.Body[:len(s.Body)-1]
			}
		}
	}

	gs.rebalanceApples()
	return dead
}

func (gs *GameState) clone() *GameState {
	c := &GameState{
		Snakes: make(map[string]*Snake),
		Apples: make([]Point, len(gs.Apples)),
		Scores: make(map[string]int),
	}
	copy(c.Apples, gs.Apples)
	for k, v := range gs.Scores {
		c.Scores[k] = v
	}
	for k, s := range gs.Snakes {
		body := make([]Point, len(s.Body))
		copy(body, s.Body)
		c.Snakes[k] = &Snake{
			Body:    body,
			Dir:     s.Dir,
			NextDir: s.NextDir,
			Growing: s.Growing,
		}
	}
	return c
}
