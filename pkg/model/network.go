package model

import (
	"fmt"
	"regexp"

	"go.uber.org/multierr"
)

//go:generate stringer -type=Network --trimprefix=Network
type Network int

const (
	// Specifies that the job does not require networking.
	NetworkNone Network = iota

	// Specifies that the job requires unfiltered raw IP networking.
	NetworkFull

	// Specifies that the job requires HTTP networking to certain domains.
	//
	// The model is: the job specifier submits a job with the domain(s) it will
	// need to communicate with, the compute provider uses this to make some
	// decision about the risk of the job and bids accordingly, and then at run
	// time the traffic is limited to only the domain(s) specified.
	//
	// As a command, something like:
	//
	//  bacalhau docker run —network=http —domain=crates.io —domain=github.com -v Qmy1234myd4t4:/code rust/compile
	//
	// The “risk” for the compute provider is that the job does something that
	// violates its terms, the terms of its hosting provider or ISP, or even the
	// law in it’s jurisdiction (e.g. accessing and spreading illegal content,
	// performing cyber attacks). So the same sort of risk as operating a Tor
	// exit node.
	//
	// The risk for the job specifier is that we are operating in an environment
	// they are paying for, so there is an incentive to hijack that environment
	// (e.g. via a compromised package download that runs a crypto miner on
	// install, and uses up all of the paid-for job time). Having the traffic
	// enforced to only domains specified makes those sorts of attacks much
	// trickier and less valuable.
	//
	// The compute provider might well enforce its limits by other means, but
	// having the domains specified up front allows it to skip bidding on jobs
	// it knows will fail in its executor. So this is hopefully a better UX for
	// job specifiers who can have their job picked up only by someone who will
	// run it successfully.
	NetworkHTTP
)

var domainRegex = regexp.MustCompile(`\b([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}\b`)

func ParseNetwork(s string) (Network, error) {
	for typ := NetworkNone; typ <= NetworkHTTP; typ++ {
		if equal(typ.String(), s) {
			return typ, nil
		}
	}

	return NetworkNone, fmt.Errorf("%T: unknown type '%s'", NetworkNone, s)
}

func (n Network) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

func (n *Network) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*n, err = ParseNetwork(name)
	return
}

type NetworkConfig struct {
	Type    Network  `json:"Type"`
	Domains []string `json:"Domains,omitempty"`
}

// Returns whether network connections should be completely disabled according
// to this config.
func (n NetworkConfig) Disabled() bool {
	return n.Type == NetworkNone
}

// IsValid returns an error if any of the fields do not pass validation, or nil
// otherwise.
func (n NetworkConfig) IsValid() (err error) {
	if n.Type < NetworkNone || n.Type > NetworkHTTP {
		err = multierr.Append(err, fmt.Errorf("invalid networking type %q", n.Type))
	}

	for _, domain := range n.Domains {
		if !domainRegex.MatchString(domain) {
			err = multierr.Append(err, fmt.Errorf("invalid domain %q", domain))
		}
	}

	return
}
