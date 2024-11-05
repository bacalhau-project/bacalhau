//go:build unit || !integration

package network_test

import (
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
)

type PortAllocatorTestSuite struct {
	suite.Suite
}

func TestPortAllocatorTestSuite(t *testing.T) {
	suite.Run(t, new(PortAllocatorTestSuite))
}

// TestGetFreePort verifies that GetFreePort returns a usable port
func (s *PortAllocatorTestSuite) TestGetFreePort() {
	port, err := network.GetFreePort()
	s.Require().NoError(err)
	s.NotEqual(0, port, "expected a non-zero port")

	// Verify we can listen on the port
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	s.Require().NoError(err)
	defer l.Close()
}

// TestPortReservation verifies that ports aren't reused within TTL
func (s *PortAllocatorTestSuite) TestPortReservation() {
	// Create allocator with 1 second TTL for testing
	allocator := network.NewPortAllocator(time.Second)

	// Get first port
	port1, err := allocator.GetFreePort()
	s.Require().NoError(err)

	// Get second port - should be different
	port2, err := allocator.GetFreePort()
	s.Require().NoError(err)
	s.NotEqual(port1, port2, "got same port within TTL period")
}

// TestConcurrentPortAllocation verifies thread-safety of port allocation
func (s *PortAllocatorTestSuite) TestConcurrentPortAllocation() {
	var wg sync.WaitGroup
	allocator := network.NewPortAllocator(time.Second)
	ports := make(map[int]bool)
	var mu sync.Mutex

	// Spawn 20 goroutines to allocate ports concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			port, err := allocator.GetFreePort()
			s.Require().NoError(err)

			mu.Lock()
			s.False(ports[port], "port %d was allocated multiple times", port)
			ports[port] = true
			mu.Unlock()

			// Verify we can listen on the port
			l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
			s.Require().NoError(err)
			l.Close()
		}()
	}
	wg.Wait()
}

// TestIsPortOpen verifies the port availability check
func (s *PortAllocatorTestSuite) TestIsPortOpen() {
	// Get a port we know should be available
	port, err := network.GetFreePort()
	s.Require().NoError(err)
	s.True(network.IsPortOpen(port), "newly allocated port should be open")

	// Listen on the port
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	s.Require().NoError(err)
	defer l.Close()

	// Port should now be in use
	s.False(network.IsPortOpen(port), "port should be in use")
}

// TestGlobalAllocator verifies that the global GetFreePort function
// prevents immediate port reuse
func (s *PortAllocatorTestSuite) TestGlobalAllocator() {
	// Get a batch of ports using the global allocator
	usedPorts := make(map[int]bool)
	for i := 0; i < 10; i++ {
		port, err := network.GetFreePort()
		s.Require().NoError(err)
		s.False(usedPorts[port], "global allocator reused port %d", port)
		usedPorts[port] = true
	}
}
