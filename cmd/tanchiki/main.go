package main

import (
	"log"
	"os"

	"ssh-games/internal/tanchiki"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":2222"
	}

	server, err := tanchiki.NewServer()
	if err != nil {
		log.Fatalf("tanchiki server: %v", err)
	}

	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
