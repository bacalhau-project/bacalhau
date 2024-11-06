package models

import (
	"os"
)

type Protocol string

const (
	// ProtocolNCLV1 is nats based async protocol based on NCL library.
	// Currently in development and not yet default protocol.
	ProtocolNCLV1 Protocol = "ncl/v1"

	// ProtocolBProtocolV2 is nats based request/response protocol.
	// Currently the default protocol while NCL is under development.
	ProtocolBProtocolV2 Protocol = "bprotocol/v2"

	// EnvPreferNCL is the environment variable to prefer NCL protocol usage.
	// This can be used to test NCL protocol while it's still in development.
	EnvPreferNCL = "BACALHAU_PREFER_NCL_PROTOCOL"
)

var (
	// preferredProtocols is the order of protocols based on preference.
	// NOTE: While NCL protocol (ProtocolNCLV1) is under active development,
	// we maintain ProtocolBProtocolV2 as the default choice for stability.
	// NCL can be enabled via BACALHAU_PREFER_NCL_PROTOCOL env var for testing
	// and development purposes. Once NCL reaches stable status, it will become
	// the default protocol.
	preferredProtocols = []Protocol{
		ProtocolBProtocolV2,
		ProtocolNCLV1,
	}
)

// String implements the Stringer interface
func (p Protocol) String() string {
	return string(p)
}

// GetPreferredProtocol accepts a slice of available protocols and returns the
// preferred protocol based on the order of preference along with any error
func GetPreferredProtocol(availableProtocols []Protocol) Protocol {
	// Check if NCL is preferred via environment variable
	if os.Getenv(EnvPreferNCL) == "true" {
		// If NCL is available when preferred, use it
		for _, p := range availableProtocols {
			if p == ProtocolNCLV1 {
				return ProtocolNCLV1
			}
		}
	}

	for _, preferred := range preferredProtocols {
		for _, available := range availableProtocols {
			if preferred == available {
				return preferred
			}
		}
	}

	// return empty string if no preferred protocol is available
	return ""
}
