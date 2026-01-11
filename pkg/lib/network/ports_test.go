package network

import (
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type PortAllocatorTestSuite struct {
	suite.Suite
}

func TestPortAllocatorTestSuite(t *testing.T) {
	suite.Run(t, new(PortAllocatorTestSuite))
}

// TestGetFreePort verifies that GetFreePort returns a usable port
func (s *PortAllocatorTestSuite) TestGetFreePort() {
	port, err := GetFreePort()
	s.Require().NoError(err)
	s.NotEqual(0, port, "expected a non-zero port")

	// Verify we can listen on the port
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	s.Require().NoError(err)
	defer func() { _ = l.Close() }()
}

// TestEvictionAndReservation tests both the TTL eviction and reservation mechanism
func (s *PortAllocatorTestSuite) TestEvictionAndReservation() {
	now := time.Now()
	allocator := &PortAllocator{
		reservedPorts: map[int]time.Time{
			8080: now.Add(-time.Second), // expired
			8081: now.Add(time.Second),  // not expired
			8082: now.Add(-time.Second), // expired
		},
		ttl:         time.Second,
		maxAttempts: 10,
	}

	// Getting a free port should clean up expired entries
	port, err := allocator.GetFreePort()
	s.Require().NoError(err)

	// Verify expired ports were cleaned up
	s.Len(allocator.reservedPorts, 2) // port we just got plus 8081
	_, hasPort := allocator.reservedPorts[8081]
	s.True(hasPort, "non-expired port should still be present")

	// New port should be reserved
	_, hasNewPort := allocator.reservedPorts[port]
	s.True(hasNewPort, "new port should be reserved")
}

// TestMaxAttempts verifies the retry limit when ports are reserved
func (s *PortAllocatorTestSuite) TestMaxAttempts() {
	allocator := &PortAllocator{
		reservedPorts: make(map[int]time.Time),
		ttl:           time.Second,
		maxAttempts:   3,
	}

	// Reserve all possible user ports (1024-65535) to force GetFreePort to fail
	// System ports (1-1023) are not used as they typically require elevated privileges
	for i := 1024; i <= 65535; i++ {
		allocator.reservedPorts[i] = time.Now().Add(time.Minute)
	}

	// Should fail after maxAttempts since all ports are reserved
	_, err := allocator.GetFreePort()
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to find an available port after 3 attempts")
}

// TestConcurrentPortAllocation verifies thread-safety of port allocation
func (s *PortAllocatorTestSuite) TestConcurrentPortAllocation() {
	var wg sync.WaitGroup
	allocator := NewPortAllocator(time.Second, 10)
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
	port, err := GetFreePort()
	s.Require().NoError(err)
	s.True(IsPortOpen(port), "newly allocated port should be open")

	// Listen on the port
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	s.Require().NoError(err)
	defer func() { _ = l.Close() }()

	// Port should now be in use
	s.False(IsPortOpen(port), "port should be in use")
}

// TestGlobalAllocator verifies the global allocator behavior
func (s *PortAllocatorTestSuite) TestGlobalAllocator() {
	usedPorts := make(map[int]bool)
	for i := 0; i < 3; i++ {
		port, err := GetFreePort()
		s.Require().NoError(err)
		s.False(usedPorts[port], "global allocator reused port %d", port)
		usedPorts[port] = true
	}
}
