//go:build unit || !integration

package network_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/stretchr/testify/suite"
)

type FreePortTestSuite struct {
	suite.Suite
}

func TestFreePortTestSuite(t *testing.T) {
	suite.Run(t, new(FreePortTestSuite))
}

func (s *FreePortTestSuite) TestGetFreePort() {
	port, err := network.GetFreePort()
	s.Require().NoError(err)
	s.NotEqual(0, port, "expected a non-zero port")

	// Try to listen on the port
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	s.Require().NoError(err)
	defer l.Close()
}

func (s *FreePortTestSuite) TestGetFreePorts() {
	count := 3
	ports, err := network.GetFreePorts(count)
	s.Require().NoError(err)
	s.Equal(count, len(ports), "expected %d ports", count)

	for _, port := range ports {
		s.NotEqual(0, port, "expected a non-zero port")

		// Try to listen on the port
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		s.Require().NoError(err, "failed to listen on newly given port")
		defer l.Close()
	}
}
