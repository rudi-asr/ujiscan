package main

import (
	"log"

	"github.com/rudi-asr/ujiscan/cmd/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
