//go:build unit || !integration

package nats

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type NATSUtilSuite struct {
	suite.Suite
	localAddresses []string
}

func TestNATSUtilSuite(t *testing.T) {
	suite.Run(t, new(NATSUtilSuite))
}

func (s *NATSUtilSuite) Test_RoutesFromStr() {
	routesStr := "nats://127.0.0.1:4222,nats://127.0.0.1:4223"
	routes, err := RoutesFromStr(routesStr, true)
	s.Require().NoError(err)
	s.Require().Len(routes, 2)
	s.Equal("nats://127.0.0.1:4222", routes[0].String())
	s.Equal("nats://127.0.0.1:4223", routes[1].String())
}

func (s *NATSUtilSuite) Test_RoutesFromStrNoLocal() {
	routesStr := "nats://127.0.0.1:4222,nats://127.0.0.1:4223"
	routes, err := RoutesFromStr(routesStr, false)
	s.Require().NoError(err)
	s.Require().Len(routes, 0)
}

func (s *NATSUtilSuite) Test_RoutesFromStrMixed() {
	// We'll use a multicast address to test here as it won't be
	// considered local (even though it is a local multicast address).
	routesStr := "nats://127.0.0.1:4222,nats://224.0.0.1:4223"
	routes, err := RoutesFromStr(routesStr, false)
	s.Require().NoError(err)
	s.Require().Len(routes, 1)
	s.Equal("nats://224.0.0.1:4223", routes[0].String())
}
