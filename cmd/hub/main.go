package main

import (
	"log"
	"os"

	"ssh-games/internal/hub"
	"ssh-games/internal/lobby"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}

	manager, err := lobby.NewManager()
	if err != nil {
		log.Fatalf("lobby manager: %v", err)
	}

	manager.PruneOrphans()

	server, err := hub.NewServer(manager)
	if err != nil {
		log.Fatalf("hub server: %v", err)
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
