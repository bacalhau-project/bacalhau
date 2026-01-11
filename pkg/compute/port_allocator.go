package compute

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// portAllocator implements the PortAllocator interface
type portAllocator struct {
	start int // First port in the dynamic allocation range
	end   int // Last port in the dynamic allocation range

	usedPorts          map[int]bool            // Tracks all allocated ports
	usedExecutionPorts map[string]map[int]bool // Maps execution ID to its allocated ports
	lastAllocated      int                     // Last successfully allocated port
	mu                 sync.Mutex              // Mutex to prevent concurrent allocation of ports
}

// NewPortAllocator creates a new port allocator with the specified port range.
// The range is only used for dynamic port allocation - static port assignments
// can use any valid port number.
func NewPortAllocator(start, end int) (PortAllocator, error) {
	err := errors.Join(
		validate.IsGreaterOrEqual(start, models.MinimumPort, "start port must be >= %d", models.MinimumPort),
		validate.IsLessOrEqual(end, models.MaximumPort, "end port must be <= %d", models.MaximumPort),
		validate.IsLessThan(start, end, "start port must be less than end port"),
	)
	if err != nil {
		return nil, err
	}

	return &portAllocator{
		start:              start,
		end:                end,
		lastAllocated:      start - 1, // Start before the range
		usedPorts:          make(map[int]bool),
		usedExecutionPorts: make(map[string]map[int]bool),
	}, nil
}

// AllocatePorts handles port allocation for a job execution. For each port mapping:
// - If a static port is specified, it is used as-is regardless of the configured range
// - If no static port is specified, a port is dynamically allocated from the configured range
// - If no target port is specified, it uses the same value as the host port
//
// Returns an error if:
// - Unable to allocate a port from the configured range
// - The execution has invalid network configuration
func (pa *portAllocator) AllocatePorts(execution *models.Execution) (models.PortMap, error) {
	networkCfg := execution.Job.Task().Network
	if networkCfg == nil || !networkCfg.Type.SupportPortAllocation() {
		return models.PortMap{}, nil
	}

	pa.mu.Lock()
	defer pa.mu.Unlock()

	allocatedPorts := make(map[int]bool)
	var portMap models.PortMap

	// Helper to cleanup on failure
	cleanup := func() {
		for port := range allocatedPorts {
			delete(pa.usedPorts, port)
		}
	}

	for _, port := range networkCfg.Ports {
		mapping := port.Copy()

		if mapping.Static == 0 {
			hostPort, err := pa.allocateDynamicPortLocked()
			if err != nil {
				cleanup()
				return nil, err
			}
			allocatedPorts[hostPort] = true
			mapping.Static = hostPort
		} else {
			if err := pa.allocateStaticPortLocked(mapping.Static); err != nil {
				cleanup()
				return nil, err
			}
			allocatedPorts[mapping.Static] = true
		}

		// For bridge mode, if no target port is specified, use the same as static port
		if networkCfg.Type == models.NetworkBridge && mapping.Target == 0 {
			mapping.Target = mapping.Static
		}

		portMap = append(portMap, mapping)
	}

	// Track ports for this execution
	if pa.usedExecutionPorts[execution.ID] == nil {
		pa.usedExecutionPorts[execution.ID] = make(map[int]bool)
	}
	for port := range allocatedPorts {
		pa.usedExecutionPorts[execution.ID][port] = true
	}

	return portMap, nil
}

// allocateStaticPortLocked checks and allocates a static port
// Caller must hold the mutex
func (pa *portAllocator) allocateStaticPortLocked(port int) error {
	// Check if port is already in use by our allocator
	if pa.usedPorts[port] {
		return fmt.Errorf("port %d is already in use", port)
	}

	// Try to actually bind to the port to ensure it's available
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) //nolint:noctx // Quick port availability check, context would require API change
	if err != nil {
		return fmt.Errorf("port %d is in use", port)
	}
	defer func() { _ = listener.Close() }()

	pa.usedPorts[port] = true
	return nil
}

// allocateDynamicPortLocked finds and allocates an available port in the configured range.
// To avoid port allocation hotspots, it uses a round-robin strategy starting from
// the last allocated port. This ensures better distribution of ports across the range
// and reduces contention when multiple allocations happen in parallel.
//
// The allocation process:
// 1. Calculates the size of the port range
// 2. Determines the starting offset based on the last allocated port
// 3. Searches through the entire range once, wrapping around using modulo arithmetic
// 4. Updates lastAllocated when a port is successfully allocated
//
// Caller must hold the mutex.
func (pa *portAllocator) allocateDynamicPortLocked() (int, error) {
	// Calculate the total number of ports in our range
	rangeSize := pa.end - pa.start + 1

	// Calculate our starting position relative to the range start
	// This gives us an offset into the range based on the last allocation
	offset := pa.lastAllocated + 1 - pa.start
	if offset < 0 || offset >= rangeSize {
		offset = 0 // Reset to start of range if we're outside it
	}

	// Single pass through the range, starting from offset
	// The modulo operation ensures we wrap around to the start when we reach the end
	for i := 0; i < rangeSize; i++ {
		port := pa.start + ((offset + i) % rangeSize)
		if pa.usedPorts[port] {
			continue
		}

		// Try to actually bind to the port
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) //nolint:noctx // Quick port availability check, context would require API change
		if err != nil {
			continue
		}
		listener.Close()

		pa.usedPorts[port] = true
		pa.lastAllocated = port
		return port, nil
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", pa.start, pa.end)
}

func (pa *portAllocator) ReleasePorts(execution *models.Execution) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	// Release all ports associated with this execution
	if ports, exists := pa.usedExecutionPorts[execution.ID]; exists {
		for port := range ports {
			delete(pa.usedPorts, port)
		}
		delete(pa.usedExecutionPorts, execution.ID)
	}
}

// compile time check for interface implementation
var _ PortAllocator = &portAllocator{}
