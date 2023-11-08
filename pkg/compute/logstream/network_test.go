//go:build unit || !integration

package logstream_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/multiformats/go-multiaddr"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (s *NetworkTestSuite) TestMultiAddrSorting() {

	type testcase struct {
		name      string
		addresses []string
		expected  []string
	}
	testcases := []testcase{
		{
			name: "simple",
			addresses: []string{
				"/ip4/127.0.0.1/tcp/2112",
				"/ip4/172.16.1.1/tcp/2112",
				"/ip6/::/tcp/2112",
				"/ip4/200.10.10.10/tcp/2112",
			},
			expected: []string{
				"/ip4/172.16.1.1/tcp/2112",
				"/ip4/200.10.10.10/tcp/2112",
				"/ip4/127.0.0.1/tcp/2112",
				"/ip6/::/tcp/2112",
			},
		},
		{
			name: "less simple",
			addresses: []string{
				"/ip4/127.0.0.1/tcp/2112",
				"/ip4/172.16.1.1/tcp/2112",
				"/ip4/172.16.2.2/tcp/2112",
				"/ip6/::/tcp/2112",
				"/ip4/200.10.10.10/tcp/2112",
			},
			expected: []string{
				// expect those in same class to maintain order
				"/ip4/172.16.1.1/tcp/2112",
				"/ip4/172.16.2.2/tcp/2112",
				"/ip4/200.10.10.10/tcp/2112",
				"/ip4/127.0.0.1/tcp/2112",
				"/ip6/::/tcp/2112",
			},
		},
		{
			name: "link-local",
			addresses: []string{
				"/ip4/169.254.1.1/tcp/2112",
				"/ip4/127.0.0.1/tcp/2112",
			},
			expected: []string{
				"/ip4/127.0.0.1/tcp/2112",
				"/ip4/169.254.1.1/tcp/2112",
			},
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			maddrs := make([]multiaddr.Multiaddr, len(tc.addresses))
			for i, addr := range tc.addresses {
				maddrs[i], _ = multiaddr.NewMultiaddr(addr)
			}

			sortedAddresses := logstream.SortAddresses(maddrs)
			actualResults := lo.Map(sortedAddresses, func(item multiaddr.Multiaddr, _ int) string {
				return item.String()
			})

			s.Require().EqualValues(tc.expected, actualResults)
		})
	}

}
