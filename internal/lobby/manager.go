package lobby

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
)

type Manager struct {
	mu          sync.RWMutex
	lobbies     map[string]*Lobby
	docker      *dockerclient.Client
	snakeImage  string
	dockerNet   string
	nextID      int
}

func NewManager() (*Manager, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	snakeImage := os.Getenv("SNAKE_IMAGE")
	if snakeImage == "" {
		snakeImage = "ssh-games-snake:latest"
	}
	dockerNet := os.Getenv("DOCKER_NETWORK")
	if dockerNet == "" {
		dockerNet = "games-net"
	}

	m := &Manager{
		lobbies:    make(map[string]*Lobby),
		docker:     cli,
		snakeImage: snakeImage,
		dockerNet:  dockerNet,
	}

	go m.cleanupLoop()
	return m, nil
}

func (m *Manager) CreateLobby(game GameType) (*Lobby, error) {
	m.mu.Lock()
	m.nextID++
	id := fmt.Sprintf("%s-%d", string(game), m.nextID)
	m.mu.Unlock()

	var image string
	switch game {
	case GameSnake:
		image = m.snakeImage
	default:
		return nil, fmt.Errorf("unknown game: %s", game)
	}

	ctx := context.Background()

	cfg := &container.Config{
		Image: image,
		Labels: map[string]string{
			"ssh-games":  "true",
			"game":       string(game),
			"lobby-id":   id,
		},
	}
	hostCfg := &container.HostConfig{
		NetworkMode: container.NetworkMode(m.dockerNet),
	}
	netCfg := &network.NetworkingConfig{}

	resp, err := m.docker.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, "lobby-"+id)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := m.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		m.docker.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("start container: %w", err)
	}

	addr, err := m.waitForContainer(ctx, resp.ID)
	if err != nil {
		m.docker.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	lob := &Lobby{
		ID:          id,
		Game:        game,
		ContainerID: resp.ID,
		Addr:        addr,
		Players:     0,
		MaxPlayers:  8,
		CreatedAt:   time.Now(),
	}

	m.mu.Lock()
	m.lobbies[id] = lob
	m.mu.Unlock()

	return lob, nil
}

func (m *Manager) waitForContainer(ctx context.Context, containerID string) (string, error) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		inspect, err := m.docker.ContainerInspect(ctx, containerID)
		if err != nil {
			return "", err
		}
		if !inspect.State.Running {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		var ip string
		for _, ep := range inspect.NetworkSettings.Networks {
			if ep.IPAddress != "" {
				ip = ep.IPAddress
				break
			}
		}
		if ip == "" {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		addr := ip + ":2222"
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			return addr, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for container")
}

func (m *Manager) GetLobby(id string) (*Lobby, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.lobbies[id]
	return l, ok
}

func (m *Manager) ListLobbies(game GameType) []LobbyInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []LobbyInfo
	for _, l := range m.lobbies {
		if l.Game == game {
			out = append(out, LobbyInfo{
				ID:         l.ID,
				Game:       l.Game,
				Players:    l.Players,
				MaxPlayers: l.MaxPlayers,
			})
		}
	}
	return out
}

func (m *Manager) IncrementPlayers(id string, delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.lobbies[id]; ok {
		l.Players += delta
	}
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanup()
	}
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	var toRemove []string
	for id, l := range m.lobbies {
		if l.Players <= 0 && time.Since(l.CreatedAt) > 60*time.Second {
			toRemove = append(toRemove, id)
		}
	}
	ids := make([]string, 0, len(toRemove))
	for _, id := range toRemove {
		ids = append(ids, m.lobbies[id].ContainerID)
		delete(m.lobbies, id)
	}
	m.mu.Unlock()

	ctx := context.Background()
	for _, cid := range ids {
		m.docker.ContainerRemove(ctx, cid, container.RemoveOptions{Force: true})
	}
}

func (m *Manager) PruneOrphans() {
	ctx := context.Background()
	f := filters.NewArgs()
	f.Add("label", "ssh-games=true")
	containers, err := m.docker.ContainerList(ctx, container.ListOptions{Filters: f})
	if err != nil {
		return
	}
	m.mu.RLock()
	known := make(map[string]bool)
	for _, l := range m.lobbies {
		known[l.ContainerID] = true
	}
	m.mu.RUnlock()

	for _, c := range containers {
		if !known[c.ID] {
			m.docker.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
			io.Discard.Write(nil)
		}
	}
}
