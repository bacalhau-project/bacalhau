package go_pinning_service_http_client

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-pinning-service-http-client/openapi"
	"github.com/multiformats/go-multiaddr"
	"time"
)

// PinGetter Getter for Pin object
type PinGetter interface {
	fmt.Stringer
	json.Marshaler
	// CID to be pinned recursively
	GetCid() cid.Cid
	// Optional name for pinned data; can be used for lookups later
	GetName() string
	// Optional list of multiaddrs known to provide the data
	GetOrigins() []string
	// Optional metadata for pin object
	GetMeta() map[string]string
}

type pinObject struct {
	openapi.Pin
}

func (p *pinObject) MarshalJSON() ([]byte, error) {
	var originsStr string
	if o := p.GetOrigins(); o != nil {
		originsBytes, err := json.Marshal(o)
		if err == nil {
			originsStr = string(originsBytes)
		}
	}

	metaStr := "{}"
	if meta := p.GetMeta(); meta != nil {
		metaBytes, err := json.Marshal(meta)
		if err == nil {
			metaStr = string(metaBytes)
		}
	}

	str := fmt.Sprintf("{ \"Cid\" : \"%v\", \"Name\" : \"%s\", \"Origins\" : %v, \"Meta\" : %v }",
		p.GetCid(), p.GetName(), originsStr, metaStr)
	return []byte(str), nil
}

func (p *pinObject) String() string {
	marshalled, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return ""
	}

	return string(marshalled)
}

func (p *pinObject) GetCid() cid.Cid {
	c, err := cid.Parse(p.Pin.Cid)
	if err != nil {
		return cid.Undef
	}
	return c
}

type Status string

const (
	StatusUnknown Status = ""
	StatusQueued  Status = Status(openapi.QUEUED)
	StatusPinning Status = Status(openapi.PINNING)
	StatusPinned  Status = Status(openapi.PINNED)
	StatusFailed  Status = Status(openapi.FAILED)
)

func (s Status) String() string {
	switch s {
	case StatusQueued, StatusPinning, StatusPinned, StatusFailed:
		return string(s)
	default:
		return string(StatusUnknown)
	}
}

var validStatuses = []Status{"queued", "pinning", "pinned", "failed"}

// PinStatusGetter Getter for Pin object with status
type PinStatusGetter interface {
	fmt.Stringer
	json.Marshaler
	// Globally unique ID of the pin request; can be used to check the status of ongoing pinning, modification of pin object, or pin removal
	GetRequestId() string
	GetStatus() Status
	// Immutable timestamp indicating when a pin request entered a pinning service; can be used for filtering results and pagination
	GetCreated() time.Time
	GetPin() PinGetter
	// List of multiaddrs designated by pinning service for transferring any new data from external peers
	GetDelegates() []multiaddr.Multiaddr
	// Optional info for PinStatus response
	GetInfo() map[string]string
}

type pinStatusObject struct {
	openapi.PinStatus
}

func (p *pinStatusObject) GetDelegates() []multiaddr.Multiaddr {
	delegates := p.PinStatus.GetDelegates()
	addrs := make([]multiaddr.Multiaddr, 0, len(delegates))
	for _, d := range delegates {
		a, err := multiaddr.NewMultiaddr(d)
		if err != nil {
			logger.Errorf("returned delegate is an invalid multiaddr: %w", err)
			continue
		}
		addrs = append(addrs, a)
	}
	return addrs
}

func (p *pinStatusObject) GetPin() PinGetter {
	return &pinObject{p.Pin}
}

func (p *pinStatusObject) GetStatus() Status {
	return Status(p.PinStatus.GetStatus())
}

func (p *pinStatusObject) GetRequestId() string {
	return p.GetRequestid()
}

func (p *pinStatusObject) MarshalJSON() ([]byte, error) {
	var delegatesStr string
	if d := p.GetDelegates(); d != nil {
		delegatesBytes, err := json.Marshal(d)
		if err == nil {
			delegatesStr = string(delegatesBytes)
		}
	}

	infoStr := "{}"
	if info := p.GetInfo(); info != nil {
		infoBytes, err := json.Marshal(info)
		if err == nil {
			infoStr = string(infoBytes)
		}
	}

	str := fmt.Sprintf("{\"Pin\" : %v, \"RequestID\" : \"%s\", \"Status\" : \"%s\", \"Created\" : \"%v\", \"Delegates\" : %v, \"Info\" : %v }",
		p.GetPin(), p.GetRequestId(), p.GetStatus(), p.GetCreated(), delegatesStr, infoStr)

	return []byte(str), nil
}

func (p *pinStatusObject) String() string {
	marshalled, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return ""
	}

	return string(marshalled)
}
