package launcher

import "net/netip"

// Backend provides a way of launching one or more challenge verifiers
type Backend interface {
	// Launch spawns a new challenge verifier for the specified flow targeting the given addresses
	Launch(id string, addresses []netip.AddrPort) error
}

// Request contains the information to start a new verification process for the local backend
type Request struct {
	// ID contains the flow identifier the request corresponds to
	ID string `json:"id"`
	// Addresses contains the IP address-port pairs to check
	Addresses []netip.AddrPort `json:"addresses"`
}
