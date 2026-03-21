package lobby

import (
	"sort"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	lobbies map[string]*Lobby
}

func NewRegistry() *Registry {
	return &Registry{
		lobbies: make(map[string]*Lobby),
	}
}

func (r *Registry) Add(l *Lobby) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lobbies[l.ID] = l
	return l.ID
}

func (r *Registry) Get(id string) (*Lobby, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.lobbies[id]
	return l, ok
}

func (r *Registry) List(game GameType) []*Lobby {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Lobby, 0, len(r.lobbies))
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

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lobbies, id)
}

func (r *Registry) IncrPlayerCount(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.lobbies[id]; ok {
		l.Players++
	}
}

func (r *Registry) DecrPlayerCount(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.lobbies[id]; ok && l.Players > 0 {
		l.Players--
	}
}
