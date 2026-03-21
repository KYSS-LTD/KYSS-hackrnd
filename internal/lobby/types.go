package lobby

import "time"

type GameType string

const (
	GameSnake         GameType = "snake"
	GamePingPong      GameType = "pingpong"
	GameFinalSentence GameType = "finalsentence"
)

type Lobby struct {
	ID          string
	Game        GameType
	ContainerID string
	Addr        string
	Players     int
	MaxPlayers  int
	CreatedAt   time.Time
}

type LobbyInfo struct {
	ID         string
	Game       GameType
	Players    int
	MaxPlayers int
}
