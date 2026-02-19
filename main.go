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
	fmt.Printf("Duel created: %p\n", duel)

	// Add 5 copies of Dark Magician (code 46986414) to player 0's hand
	for i := 0; i < 5; i++ {
		bridge.AddCard(duel, 46986414, 0, bridge.LOC_HAND, uint32(i), bridge.POS_FACEUP)
	}

	// Add 5 copies to player 0's deck so the engine doesn't immediately error
	for i := 0; i < 40; i++ {
		bridge.AddCard(duel, 46986414, 0, bridge.LOC_DECK, uint32(i), bridge.POS_FACEDOWN)
	}

	// Same for player 1
	for i := 0; i < 40; i++ {
		bridge.AddCard(duel, 46986414, 1, bridge.LOC_DECK, uint32(i), bridge.POS_FACEDOWN)
	}

	bridge.StartDuel(duel)
	fmt.Println("Duel started, processing...")

	// Step through the engine loop
	for i := 0; i < 20; i++ {
		status := bridge.ProcessDuel(duel)
		fmt.Printf("  step %2d: status=%d ", i, status)
		switch status {
		case bridge.DUEL_STATUS_END:
			fmt.Println("(END)")
			goto done
		case bridge.DUEL_STATUS_AWAITING:
			fmt.Println("(AWAITING — engine wants a decision)")
			msg := bridge.GetMessage(duel)
			if len(msg) > 0 && bridge.MessageType(msg[0]) == bridge.MSG_SELECT_IDLECMD {
				idle := bridge.ParseIdleCmd(msg)
				fmt.Printf("  Player %d's turn\n", idle.Player)
				fmt.Printf("  Normal summons available: %d\n", len(idle.Summons))
				fmt.Printf("  Spell/trap sets available: %d\n", len(idle.SZoneSetCards))
				fmt.Printf("  Activations available: %d\n", len(idle.Activations))
				fmt.Printf("  Can go to battle phase: %v\n", idle.CanBattlePhase)
				fmt.Printf("  Can end turn: %v\n", idle.CanEndPhase)
				for i, s := range idle.Summons {
					fmt.Printf("    summon[%d]: code=%d loc=%d seq=%d\n", i, s.CardCode, s.Location, s.Sequence)
				}
			}
			goto done
		case bridge.DUEL_STATUS_CONTINUE:
			fmt.Println("(CONTINUE)")
		}
	}
done:
	fmt.Println("Done.")
}
