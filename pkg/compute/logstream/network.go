package logstream

import (
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/samber/lo"
)

type AddressType int

const (
	PrivateAddress  = 0
	PublicAddress   = 1
	LoopbackAddress = 2
	Unspecified     = 3
	LinkLocal       = 4
)

func findTCPAddress(host host.Host) string {
	peerID := host.ID().Pretty()
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID))

	addresses := SortAddresses(host.Addrs())

	for _, addr := range addresses {
		for _, protocol := range addr.Protocols() {
			if protocol.Name == "tcp" {
				return addr.Encapsulate(hostAddr).String()
			}
		}
	}

	// If we can't find any TCP records, then we'll try with the first record
	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func getIPForAddress(addr multiaddr.Multiaddr) net.IP {
	ip, err := addr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		ip, _ = addr.ValueForProtocol(multiaddr.P_IP6)
	}
	return net.ParseIP(ip)
}

func SortAddresses(addresses []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	grouped := lo.GroupBy[multiaddr.Multiaddr, AddressType](addresses, func(item multiaddr.Multiaddr) AddressType {
		ip := getIPForAddress(item)
		if ip.IsLoopback() {
			return LoopbackAddress
		} else if ip.IsPrivate() {
			return PrivateAddress
		} else if ip.IsUnspecified() {
			return Unspecified
		} else if ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
			return LinkLocal
		}

		return PublicAddress
	})

	sorted := make([]multiaddr.Multiaddr, 0, len(addresses))
	sorted = append(sorted, grouped[PrivateAddress]...)
	sorted = append(sorted, grouped[PublicAddress]...)
	sorted = append(sorted, grouped[LoopbackAddress]...)
	sorted = append(sorted, grouped[Unspecified]...)
	return append(sorted, grouped[LinkLocal]...)
}
