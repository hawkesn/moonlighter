//go:build !windows

package moonlighter

import "errors"

// ReadSignals fails closed off Windows: the owner-presence signals have no
// portable reading, so nothing is lent rather than lending blindly.
func ReadSignals() Signals {
	return Signals{Err: errors.New("owner signals need Windows")}
}
