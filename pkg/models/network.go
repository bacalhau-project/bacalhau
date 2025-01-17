//go:generate stringer -type=Network --trimprefix=Network
package models

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

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
	//  bacalhau docker run —network=http —domain=crates.io —domain=github.com -i ipfs://Qmy1234myd4t4,dst=/code rust/compile
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
		if strings.EqualFold(typ.String(), strings.TrimSpace(s)) {
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
func (n *NetworkConfig) Disabled() bool {
	return n.Type == NetworkNone
}

// Normalize ensures that the network config is in a consistent state.
func (n *NetworkConfig) Normalize() {
	if n == nil {
		return
	}
	// Ensure that an empty and nil slice are treated the same
	if len(n.Domains) == 0 {
		n.Domains = make([]string, 0)
	}
	// Ensure that domains are lowercased, and trimmed of whitespace
	for i, domain := range n.Domains {
		n.Domains[i] = strings.TrimSpace(strings.ToLower(domain))
	}
}

func (n *NetworkConfig) Copy() *NetworkConfig {
	if n == nil {
		return nil
	}
	return &NetworkConfig{
		Type:    n.Type,
		Domains: slices.Clone(n.Domains),
	}
}

// Validate returns an error if any of the fields do not pass validation, or nil
// otherwise.
func (n *NetworkConfig) Validate() (err error) {
	if n.Type < NetworkNone || n.Type > NetworkHTTP {
		err = errors.Join(err, fmt.Errorf("invalid networking type %q", n.Type))
	}

	// TODO(forrest): should return an error if the network type is not HTTP and domains are set.
	for _, domain := range n.Domains {
		if domainRegex.MatchString(domain) {
			continue
		}
		if net.ParseIP(domain) != nil {
			continue
		}
		err = errors.Join(err, fmt.Errorf("invalid domain %q", domain))
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
func (n *NetworkConfig) DomainSet() []string {
	if n.Domains == nil {
		return []string{}
	}
	domains := slices.Clone(n.Domains)

	// Can use cmp package in go 1.21, but for now...
	cmp := func(a int, b int) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}

	// Compacts the slice by removing any elements that are subdomains of other
	// elements. This previously used slices.CompactFunc but that only runs
	// pairwise and so if we had
	// [foo.com, x.foo.com, y.foo.com] then post compact we would have
	// [foo.com, y.foo.com] which is not what we want.
	// This version of compact will keep compacting until no changes were
	// made in the last iteration, at which point we will filter out any
	// empty strings
	compact := func(domains []string) []string {
		pre := len(domains)
		post := 0

		// If there's been no change in length, then we can safely return
		for pre != post {
			pre = len(domains)
			domains = slices.CompactFunc(domains, func(a, b string) bool {
				return matchDomain(a, b) == 0
			})
			post = len(domains)
		}

		return lo.Filter[string](domains, func(item string, _ int) bool {
			return item != ""
		})
	}

	slices.SortFunc(domains, func(a, b string) int {
		// If the domains "match", the match may be the result of a wildcard. We
		// want to keep the wildcard because it matches more things. Wildcards
		// will always be shorter than any subdomain they match, so we can
		// simply sort on string length. Compact will then remove non-wildcards.
		ret := matchDomain(a, b)
		if ret == 0 {
			return cmp(len(a), len(b))
		} else {
			return strings.Compare(a, b)
		}
	})

	return compact(domains)
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

	lCur, rCur := len(lefts)-1, len(rights)-1
	for lCur >= 0 && rCur >= 0 {
		// If neither is a blank, these components need to match.
		if lefts[lCur] != wildcard && rights[rCur] != wildcard {
			if diff = strings.Compare(lefts[lCur], rights[rCur]); diff != 0 {
				return diff
			}
		}

		// If both are blanks, they match.
		if lefts[lCur] == wildcard || rights[rCur] == wildcard {
			break
		}

		// Blank means we are matching any subdomains, so only the rest of
		// the domain needs to match for this to work.
		if lefts[lCur] != wildcard {
			lCur -= 1
		}

		if rights[rCur] != wildcard {
			rCur -= 1
		}
	}

	// If we are here, we have run out of components; either the domains match
	// in all components or one of them is a wildcard.
	return 0
}
