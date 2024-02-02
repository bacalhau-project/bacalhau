//go:build unit || !integration

package network_test

import (
	"net"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func TestEnvTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func test_addresses() []string {
	return []string{
		"127.0.0.1",       // localhost
		"169.254.46.5",    // link local
		"192.168.0.1",     // private
		"10.10.10.10",     // private
		"172.16.20.24",    // private
		"225.255.255.255", // multicast
		"86.13.65.80",     //public
	}
}

func AddressList() ([]net.IP, error) {
	addr := test_addresses()
	result := make([]net.IP, len(addr))

	for i := range addr {
		result[i] = net.ParseIP(addr[i])
	}

	return result, nil
}

func (s *NetworkTestSuite) TestFetch() {
	type testcase struct {
		name        string
		addressType network.AddressType
		expected    []string
	}
	testcases := []testcase{
		{"localhost", network.LoopbackAddress, []string{"127.0.0.1"}},
		{"linklocal", network.LinkLocal, []string{"169.254.46.5"}},
		{"multicast", network.Multicast, []string{"225.255.255.255"}},
		{"private", network.PrivateAddress, []string{"192.168.0.1", "10.10.10.10", "172.16.20.24"}},
		{"public", network.PublicAddress, []string{"86.13.65.80"}},
		{"any", network.Any, test_addresses()},
	}

	for t := range testcases {
		tc := testcases[t]
		s.Run(tc.name, func() {
			addr, err := network.GetNetworkAddress(tc.addressType, AddressList)
			s.Require().NoError(err)
			s.Require().ElementsMatch(tc.expected, addr)
		})
	}
}
