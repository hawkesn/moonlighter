package main

import (
	"fmt"
	"os"

	"github.com/hawkesn/moonlighter"
)

// check prints the owner-presence signals; see lend.go for what they stop.
func main() {
	if len(os.Args) != 2 || os.Args[1] != "check" {
		fmt.Fprintln(os.Stderr, "usage: moonlighter check")
		os.Exit(2)
	}
	sig := moonlighter.ReadSignals()
	fmt.Print(moonlighter.FormatSignals(sig))
	if sig.Err != nil {
		os.Exit(1)
	}
}
