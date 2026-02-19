package main

import (
	"fmt"
	"os"

	"github.com/shawn/ygo-visualizer/bridge"
)

func main() {
	fmt.Fprintln(os.Stderr, "starting...")
	major, minor := bridge.GetVersion()
	fmt.Fprintf(os.Stderr, "OCG Core version: %d.%d\n", major, minor)
}
