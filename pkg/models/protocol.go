package models

type Protocol string

const (
	// ProtocolNCLV1 is nats based async protocol based on NCL library.
	// Currently in development and not yet default protocol.
	ProtocolNCLV1 Protocol = "ncl/v1"

	// ProtocolBProtocolV2 is nats based request/response protocol.
	// Currently the default protocol while NCL is under development.
	ProtocolBProtocolV2 Protocol = "bprotocol/v2"
)

var (
	// preferredProtocols is the order of protocols based on preference.
	preferredProtocols = []Protocol{
		ProtocolNCLV1,
		ProtocolBProtocolV2,
	}
)

// String implements the Stringer interface
func (p Protocol) String() string {
	return string(p)
}

// GetPreferredProtocol accepts a slice of available protocols and returns the
// preferred protocol based on the order of preference along with any error
func GetPreferredProtocol(availableProtocols []Protocol) Protocol {
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
