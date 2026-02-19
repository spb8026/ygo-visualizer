package main

import (
	"fmt"
	"os"

	"github.com/spb8026/ygo-visualizer/bridge"
)

func main() {
	major, minor := bridge.GetVersion()
	fmt.Printf("OCG Core version: %d.%d\n", major, minor)

	duel, err := bridge.CreateDuel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create duel: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Duel created successfully: %p\n", duel)
}
