package multiaddresses

import (
	"sort"

	"github.com/multiformats/go-multiaddr"
	"golang.org/x/exp/slices"
)

func SortLocalhostFirst(multiAddresses []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	multiAddresses = slices.Clone(multiAddresses)
	preferLocalhost := func(m multiaddr.Multiaddr) int {
		count := 0
		if _, err := m.ValueForProtocol(multiaddr.P_TCP); err == nil {
			count++
		}
		if ip, err := m.ValueForProtocol(multiaddr.P_IP4); err == nil {
			count++
			if ip == "127.0.0.1" {
				count++
			}
		} else if ip, err := m.ValueForProtocol(multiaddr.P_IP6); err == nil && ip != "::1" {
			count++
		}
		return count
	}
	sort.Slice(multiAddresses, func(i, j int) bool {
		return preferLocalhost(multiAddresses[i]) > preferLocalhost(multiAddresses[j])
	})

	return multiAddresses
}
