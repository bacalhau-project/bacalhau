package v1beta1

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"go.uber.org/multierr"
	"golang.org/x/exp/slices"
)

//go:generate stringer -type=Network --trimprefix=Network
type Network int

const (
	// NetworkNone specifies that the job does not require networking.
	NetworkNone Network = iota

	// NetworkFull specifies that the job requires unfiltered raw IP networking.
	NetworkFull

	// NetworkHTTP specifies that the job requires HTTP networking to certain domains.
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
	// law in its jurisdiction (e.g. accessing and spreading illegal content,
	// performing cyberattacks). So the same sort of risk as operating a Tor
	// exit node.
	//
	// The risk for the job specifier is that we are operating in an environment
	// they are paying for, so there is an incentive to hijack that environment
	// (e.g. via a compromised package download that runs a crypto miner on
	// install, and uses up all the paid-for job time). Having the traffic
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

// Disabled returns whether network connections should be completely disabled according
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
		if domainRegex.MatchString(domain) {
			continue
		}
		if net.ParseIP(domain) != nil {
			continue
		}
		err = multierr.Append(err, fmt.Errorf("invalid domain %q", domain))
	}

	return
}

// DomainSet returns the "unique set" of domains from the network config.
// Domains listed multiple times and any subdomain that is also matched by a
// wildcard is removed.
//
// This is something of an implementation detail – it matches the behavior
// expected by our Docker HTTP gateway, which complains and/or fails to start if
// these requirements are not met.
func (n NetworkConfig) DomainSet() []string {
	domains := slices.Clone(n.Domains)
	slices.SortFunc(domains, func(a, b string) bool {
		// If the domains "match", the match may be the result of a wildcard. We
		// want to keep the wildcard because it matches more things. Wildcards
		// will always be shorter than any subdomain they match, so we can
		// simply sort on string length. Compact will then remove non-wildcards.
		ret := matchDomain(a, b)
		if ret == 0 {
			return len(a) < len(b)
		} else {
			return ret < 0
		}
	})
	domains = slices.CompactFunc(domains, func(a, b string) bool {
		return matchDomain(a, b) == 0
	})
	return domains
}

func matchDomain(left, right string) (diff int) {
	const wildcard = ""
	lefts := strings.Split(strings.ToLower(strings.Trim(left, " ")), ".")
	rights := strings.Split(strings.ToLower(strings.Trim(right, " ")), ".")

	diff = len(lefts) - len(rights)
	if diff != 0 && lefts[0] != wildcard && rights[0] != wildcard {
		// Domains don't have same number of components, so
		// the one that is longer should sort after.
		return diff
	}

	lcur, rcur := len(lefts)-1, len(rights)-1
	for lcur >= 0 && rcur >= 0 {
		// If neither is a blank, these components need to match.
		if lefts[lcur] != wildcard && rights[rcur] != wildcard {
			if diff = strings.Compare(lefts[lcur], rights[rcur]); diff != 0 {
				return diff
			}
		}

		// If both are blanks, they match.
		if lefts[lcur] == wildcard || rights[rcur] == wildcard {
			break
		}

		// Blank means we are matching any subdomains, so only the rest of
		// the domain needs to match for this to work.
		if lefts[lcur] != wildcard {
			lcur -= 1
		}

		if rights[rcur] != wildcard {
			rcur -= 1
		}
	}

	// If we are here, we have run out of components; either the domains match
	// in all components or one of them is a wildcard.
	return 0
}
