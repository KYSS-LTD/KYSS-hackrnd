package main

import (
	"log"
	"os"

	"ssh-games/internal/pingpong"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}

	server, err := pingpong.NewServer()
	if err != nil {
		log.Fatalf("ping pong server: %v", err)
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
