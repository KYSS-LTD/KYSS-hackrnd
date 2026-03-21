package main

import (
	"log"
	"os"

	"ssh-games/internal/kitchen"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}
	server, err := kitchen.NewServer()
	if err != nil {
		log.Fatalf("kitchen server: %v", err)
	}
	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
