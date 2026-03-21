package main

import (
	"log"
	"os"

	"ssh-games/internal/finalsentence"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}

	server, err := finalsentence.NewServer()
	if err != nil {
		log.Fatalf("final sentence server: %v", err)
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
