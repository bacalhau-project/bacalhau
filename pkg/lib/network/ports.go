package network

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	defaultPortAllocatorTTL = 5 * time.Second
	defaultMaxAttempts      = 10
)

// PortAllocator manages thread-safe allocation of network ports with a time-based reservation system.
// Once a port is allocated, it won't be reallocated until after the TTL expires, helping prevent
// race conditions in concurrent port allocation scenarios.
type PortAllocator struct {
	mu            sync.Mutex
	reservedPorts map[int]time.Time
	ttl           time.Duration
	maxAttempts   uint
}

var (
	// globalAllocator is the package-level port allocator instance used by GetFreePort
	globalAllocator = NewPortAllocator(defaultPortAllocatorTTL, defaultMaxAttempts)
)

// NewPortAllocator creates a new PortAllocator instance.
// ttl determines how long a port remains reserved after allocation.
// maxAttempts limits how many times we'll try to find an unreserved port before giving up.
func NewPortAllocator(ttl time.Duration, maxAttempts uint) *PortAllocator {
	if maxAttempts == 0 {
		maxAttempts = defaultMaxAttempts
	}
	return &PortAllocator{
		reservedPorts: make(map[int]time.Time),
		ttl:           ttl,
		maxAttempts:   maxAttempts,
	}
}

// GetFreePort returns an available port using the global port allocator.
// The returned port is guaranteed to not be reallocated by this package
// for the duration of the TTL (default 5 seconds).
func GetFreePort() (int, error) {
	return globalAllocator.GetFreePort()
}

// GetFreePort returns an available port and reserves it for the duration of the TTL.
// If a port is already reserved but its TTL has expired, it may be returned if it's
// still available on the system. Returns error if unable to find an available port
// after maxAttempts tries.
func (pa *PortAllocator) GetFreePort() (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Clean up expired reservations eagerly to prevent memory growth
	now := time.Now()
	for port, expiration := range pa.reservedPorts {
		if now.After(expiration) {
			delete(pa.reservedPorts, port)
		}
	}

	for attempts := uint(0); attempts < pa.maxAttempts; attempts++ {
		port, err := getFreePortFromSystem()
		if err != nil {
			return 0, fmt.Errorf("failed to get port from system: %w", err)
		}

		if _, reserved := pa.reservedPorts[port]; !reserved {
			pa.reservedPorts[port] = now.Add(pa.ttl)
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to find an available port after %d attempts", pa.maxAttempts)
}

// getFreePortFromSystem asks the operating system for an available port by creating
// a TCP listener with port 0, which triggers the OS to assign a random available port.
//
// Essentially the same code as https://github.com/phayes/freeport but we bind
// to 0.0.0.0 to ensure the port is free on all interfaces, and not just localhost.GetFreePort
// Ports must be unique for an address, not an entire system and so checking just localhost
// is not enough.
func getFreePortFromSystem() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// IsPortOpen checks if a specific port is available for use by attempting to create
// a TCP listener on that port. It returns true if the port is available, false otherwise.
// The caller should note that the port's availability may change immediately after
// this check returns.
func IsPortOpen(port int) bool {
	addr := net.JoinHostPort("", strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr) //nolint:noctx // Simple utility function, context would require API change
	if err != nil {
		return false
	}
	_ = _ = ln.Close()
	return true
}
