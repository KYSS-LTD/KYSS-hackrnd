package snake

const (
	Width  = 16
	Height = 9
)

type Point struct {
	X, Y int
}

func (p Point) Equal(o Point) bool {
	return p.X == o.X && p.Y == o.Y
}

var (
	DirUp    = Point{0, -1}
	DirDown  = Point{0, 1}
	DirLeft  = Point{-1, 0}
	DirRight = Point{1, 0}
)

func opposite(d Point) Point {
	return Point{-d.X, -d.Y}
}

type Snake struct {
	Body    []Point
	Dir     Point
	NextDir Point
	Growing int
}

func (s *Snake) Head() Point {
	return s.Body[0]
}

func (s *Snake) OccupiesExcludeHead(p Point) bool {
	for _, b := range s.Body[1:] {
		if b.Equal(p) {
			return true
		}
	}
	return false
}

func (s *Snake) Occupies(p Point) bool {
	for _, b := range s.Body {
		if b.Equal(p) {
			return true
		}
	}
	return false
}

type GameState struct {
	Snakes map[string]*Snake
	Apples []Point
	Scores map[string]int
}

func (s *Snake) Len() int {
	return len(s.Body)
}
