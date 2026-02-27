package duelInterface

import (
	"fmt"
	"os"

	"github.com/spb8026/ygo-visualizer/bridge"
)

func RunCLI() {
	duel, err := bridge.NewDuel(bridge.DuelOptions{
		Seed:              [4]uint64{12345, 0, 0, 0},
		StartingLP:        8000,
		StartingDrawCount: 5,
		DrawCountPerTurn:  1,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create duel: %v\n", err)
		os.Exit(1)
	}
	defer duel.Close()

	const GeminiElf = uint32(69140098)
	for i := 0; i < 40; i++ {
		duel.AddCard(0, 0, GeminiElf, 0, bridge.LOC_DECK, uint32(i), bridge.POS_FACEDOWN)
		duel.AddCard(1, 0, GeminiElf, 1, bridge.LOC_DECK, uint32(i), bridge.POS_FACEDOWN)
	}

	duel.Start()

	status, msgs, err := duel.Step()
	fmt.Printf("Status: %d, Messages: %d, Error: %v\n", status, len(msgs), err)
}
