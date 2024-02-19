package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
)

type AddressType int

const (
	PrivateAddress AddressType = iota
	PublicAddress
	LoopbackAddress
	LinkLocal
	Multicast
	Any
)

type AddressLister func() ([]net.IP, error)

func AllAddresses() ([]net.IP, error) {
	var result []net.IP

	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get interface addresses")
	}

	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse address: %s", address.String()))
		}

		ipAs4 := ip.To4()
		if ipAs4 != nil {
			result = append(result, ipAs4)
		}
	}

	return result, nil
}

// GetNetworkAddress returns a list of network addresses of the requested type,
// sourcing the addresses from the provided AddressLister. It is expected that
// network.AddAddresses() will be the default address lister. The result is
// a list of strings representing the network addresses in the order they
// were returned by the AddressLister.
func GetNetworkAddress(requested AddressType, getAddresses AddressLister) ([]string, error) {
	addresses, err := getAddresses()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(addresses))

	for i := range addresses {
		addr := addresses[i]

		switch requested {
		case Any:
			result = append(result, addr.String())
		case PrivateAddress:
			if addr.IsPrivate() || isCarrierGradeNAT(addr) {
				result = append(result, addr.String())
			}
		case PublicAddress:
			if !addr.IsPrivate() && addr.IsGlobalUnicast() && !isCarrierGradeNAT(addr) {
				result = append(result, addr.String())
			}
		case LoopbackAddress:
			if addr.IsLoopback() {
				result = append(result, addr.String())
			}
		case LinkLocal:
			if addr.IsLinkLocalMulticast() || addr.IsLinkLocalUnicast() {
				result = append(result, addr.String())
			}
		case Multicast:
			if isMulticastAddress(addr) {
				result = append(result, addr.String())
			}
		}
	}

	return result, nil
}

func (a AddressType) String() string {
	switch a {
	case PrivateAddress:
		return "private"
	case PublicAddress:
		return "public"
	case LoopbackAddress:
		return "loopback"
	case LinkLocal:
		return "linklocal"
	case Multicast:
		return "multicast"
	case Any:
		return "any"
	default:
		return "unknown"
	}
}

func AddressTypeFromString(t string) (AddressType, bool) {
	switch strings.ToLower(t) {
	case "private":
		return PrivateAddress, true
	case "public":
		return PublicAddress, true
	case "loopback", "localhost", "local":
		return LoopbackAddress, true
	case "linklocal":
		return LinkLocal, true
	case "multicast":
		return Multicast, true
	case "any":
		return Any, true
	default:
		return Any, false
	}
}

func isCarrierGradeNAT(addr net.IP) bool {
	_, net, _ := net.ParseCIDR("100.64.0.0/10")
	return net.Contains(addr)
}

func isMulticastAddress(addr net.IP) bool {
	_, net, _ := net.ParseCIDR("224.0.0.0/4")
	return net.Contains(addr)
}
