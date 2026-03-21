package lobby

import (
	"sort"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	lobbies map[int]*Lobby
	nextID  int
}

func NewRegistry() *Registry {
	return &Registry{
		lobbies: make(map[int]*Lobby),
		nextID:  1,
	}
}

func (r *Registry) Add(l *Lobby) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	l.ID = r.nextID
	r.lobbies[r.nextID] = l
	r.nextID++
	return l.ID
}

func (r *Registry) Get(id int) (*Lobby, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.lobbies[id]
	return l, ok
}

func (r *Registry) List(game GameType) []*Lobby {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Lobby
	for _, l := range r.lobbies {
		if l.Game == game {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (r *Registry) Remove(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lobbies, id)
}

func (r *Registry) IncrPlayerCount(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.lobbies[id]; ok {
		l.PlayerCount++
		l.State = LobbyStatePlaying
	}
}

func (r *Registry) DecrPlayerCount(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.lobbies[id]; ok {
		if l.PlayerCount > 0 {
			l.PlayerCount--
		}
		if l.PlayerCount == 0 {
			l.State = LobbyStateWaiting
		}
	}
}
