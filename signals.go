package moonlighter

import (
	"fmt"
	"time"
)

// IdleSince converts the tick count and the last-input tick into the time
// since the owner last input; unsigned subtraction handles the tick wrap.
func IdleSince(tick, last uint32) time.Duration {
	return time.Duration(tick-last) * time.Millisecond
}

// FullScreenGame reports whether the shell notification state means a
// full-screen application has the owner's attention: busy, Direct3D full
// screen or presentation mode.
func FullScreenGame(state uint32) bool {
	return state >= 2 && state <= 4
}

// FormatSignals prints each signal on its own line, and any error last.
func FormatSignals(sig Signals) string {
	out := fmt.Sprintf("idle: %v\ngame: %v\nfree_ram: %d MiB\n",
		sig.Idle, sig.OwnerInput, sig.FreeRAM>>20)
	if sig.Err != nil {
		out += "error: " + sig.Err.Error() + "\n"
	}
	return out
}
