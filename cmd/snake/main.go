package main

import (
	"log"
	"os"

	"ssh-games/internal/snake"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}

	server, err := snake.NewServer()
	if err != nil {
		log.Fatalf("snake server: %v", err)
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
