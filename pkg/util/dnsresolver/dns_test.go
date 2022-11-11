//go:build !integration

package dnsresolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DNSResolverSuite struct {
	suite.Suite
}

func TestDNSResolver(t *testing.T) {
	suite.Run(t, new(DNSResolverSuite))
}

func (s *DNSResolverSuite) TestLookup() {

	testcases := []struct {
		name        string
		domainname  string
		shouldError bool
	}{
		{name: "Docker", domainname: "docker.io", shouldError: false},
		{name: "BadDomain", domainname: "bad.domain", shouldError: true},
	}

	for _, tc := range testcases {
		ctx := context.TODO()
		_, err := LookupIP(ctx, tc.domainname, 5)
		require.True(s.T(), tc.shouldError == (err != nil), tc.name)
	}

}
