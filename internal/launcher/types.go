package launcher

import "net/netip"

// Backend provides a way of launching one or more challenge verifiers
type Backend interface {
	// Launch spawns a new challenge verifier for the specified flow targeting the given addresses
	Launch(id string, addresses []netip.AddrPort) error
}
