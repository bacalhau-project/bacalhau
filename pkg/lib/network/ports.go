package network

import (
	"net"
	"strconv"
	"sync"
	"time"
)

const defaultPortAllocatorTTL = 5 * time.Second

// PortAllocator manages thread-safe allocation of network ports with a time-based reservation system.
// Once a port is allocated, it won't be reallocated until after the TTL expires, helping prevent
// race conditions in concurrent port allocation scenarios.
type PortAllocator struct {
	mu            sync.Mutex
	reservedPorts map[int]time.Time
	ttl           time.Duration
}

var (
	// globalAllocator is the package-level port allocator instance used by GetFreePort
	globalAllocator = NewPortAllocator(defaultPortAllocatorTTL)
)

// NewPortAllocator creates a new PortAllocator instance with the specified TTL.
// The TTL determines how long a port remains reserved after allocation.
func NewPortAllocator(ttl time.Duration) *PortAllocator {
	return &PortAllocator{
		reservedPorts: make(map[int]time.Time),
		ttl:           ttl,
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
// still available on the system.
func (pa *PortAllocator) GetFreePort() (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	port, err := getFreePortFromSystem()
	if err != nil {
		return 0, err
	}

	// Keep trying until we find a port that isn't reserved or has expired reservation
	now := time.Now()
	for {
		if expiration, reserved := pa.reservedPorts[port]; !reserved || now.After(expiration) {
			break
		}
		port, err = getFreePortFromSystem()
		if err != nil {
			return 0, err
		}
	}

	pa.reservedPorts[port] = now.Add(pa.ttl)
	return port, nil
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
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// IsPortOpen checks if a specific port is available for use by attempting to create
// a TCP listener on that port. It returns true if the port is available, false otherwise.
// The caller should note that the port's availability may change immediately after
// this check returns.
func IsPortOpen(port int) bool {
	addr := net.JoinHostPort("", strconv.Itoa(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
