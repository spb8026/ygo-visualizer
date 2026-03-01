package duelInterface

import (
	"fmt"
	"os"

	"github.com/spb8026/ygo-visualizer/bridge"
	duelpb "github.com/spb8026/ygo-visualizer/ygopenpb"
	"google.golang.org/protobuf/proto"
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
	for {
		status, msgs, err := duel.Step()
		fmt.Printf("Status: %d, Messages: %d, Error: %v\n", status, len(msgs), err)

		for _, b := range msgs {
			var m duelpb.Msg
			if err := proto.Unmarshal(b, &m); err != nil {
				fmt.Printf("  decode error: %v\n", err)
				continue
			}
			fmt.Printf("  Msg: %s\n", m.String())
			if sel := m.GetRequest().GetSelectIdle(); sel != nil {
				err := duel.SendIdlePhase(sel.GetAvailablePhase())
				if err != nil {
					fmt.Printf("  answer error: %v\n", err)
				} else {
					fmt.Printf("  sent phase answer: %d\n", sel.GetAvailablePhase())
				}
			}
			if m.GetRequest().GetSelectToChain() != nil {
				err := duel.SendSelectToChainNoOp()
				if err != nil {
					fmt.Printf("  answer error: %v\n", err)
				} else {
					fmt.Printf("  sent no-op chain answer\n")
				}
			}
		}

		var input string
		fmt.Scanln(&input)
		if input == "q" {
			break
		}
	}
}
