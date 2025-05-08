package main

import (
	"log"

	"murmapp.hook/internal"
)

func main() {
	if err := internal.Run(); err != nil {
		log.Fatalf("❌ fatal error: %v", err)
	}
}
