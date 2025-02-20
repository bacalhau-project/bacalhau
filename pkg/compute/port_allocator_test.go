//go:build unit || !integration

package compute_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/stretchr/testify/suite"
)

type PortAllocatorTestSuite struct {
	suite.Suite
}

func TestPortAllocatorTestSuite(t *testing.T) {
	suite.Run(t, new(PortAllocatorTestSuite))
}

func (s *PortAllocatorTestSuite) TestNoNetworkConfigReturnsEmptyMappings() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	execution := mock.Execution()
	mappings, err := pa.AllocatePorts(execution)
	s.Require().NoError(err)
	s.Empty(mappings)
}

func (s *PortAllocatorTestSuite) TestNetworkTypeNoneReturnsEmptyMappings() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkNone,
	}

	mappings, err := pa.AllocatePorts(execution)
	s.Require().NoError(err)
	s.Empty(mappings)
}

func (s *PortAllocatorTestSuite) TestAllocatesPortsWithStaticAndTarget() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Target: 80},                 // Dynamic host port, container port 80
			{Static: 6000, Target: 8080}, // Static host port outside range, container port 8080
			{Static: 1500, Target: 9090}, // Static host port inside range, container port 9090
			{Target: 5000},               // Dynamic host port, container port 5000
		},
	}

	mappings, err := pa.AllocatePorts(execution)
	s.Require().NoError(err)
	s.Len(mappings, 4)

	// First port should be auto-allocated in range
	s.GreaterOrEqual(int(mappings[0].Static), 3000)
	s.LessOrEqual(int(mappings[0].Static), 4000)
	s.Equal(80, mappings[0].Target)

	// Second port should keep static port even though outside range
	s.Equal(6000, mappings[1].Static)
	s.Equal(8080, mappings[1].Target)

	// Third port should keep static port inside range
	s.Equal(1500, mappings[2].Static)
	s.Equal(9090, mappings[2].Target)

	// Fourth port should be auto-allocated in range
	s.GreaterOrEqual(int(mappings[3].Static), 3000)
	s.LessOrEqual(int(mappings[3].Static), 4000)
	s.Equal(5000, mappings[3].Target)
}

func (s *PortAllocatorTestSuite) TestValidatesPortAllocatorRange() {
	_, err := compute.NewPortAllocator(4000, 3000) // Invalid range
	s.Error(err)
	s.Contains(err.Error(), "start port must be less than end port")
}

func (s *PortAllocatorTestSuite) TestFailsOnPortReuse() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	// First execution requests port 1500
	execution1 := mock.Execution()
	execution1.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Static: 1500, Target: 80},
		},
	}

	mappings, err := pa.AllocatePorts(execution1)
	s.Require().NoError(err)
	s.Len(mappings, 1)
	s.Equal(1500, mappings[0].Static)

	// Second execution requests same port
	execution2 := mock.Execution()
	execution2.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Static: 1500, Target: 80},
		},
	}

	_, err = pa.AllocatePorts(execution2)
	s.Error(err)
	s.Contains(err.Error(), "port 1500 is already in use")
}

func (s *PortAllocatorTestSuite) TestPortRelease() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	// Allocate ports for an execution
	execution := mock.Execution()
	execution.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Static: 1500, Target: 80},
			{Target: 8080}, // Dynamic port
		},
	}

	mappings, err := pa.AllocatePorts(execution)
	s.Require().NoError(err)
	s.Len(mappings, 2)

	// Remember the dynamic port allocated
	dynamicPort := mappings[1].Static

	// Release the ports
	pa.ReleasePorts(execution)

	// Should be able to allocate the same ports again
	execution2 := mock.Execution()
	execution2.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Static: 1500, Target: 80},
			{Static: dynamicPort, Target: 8080},
		},
	}

	mappings2, err := pa.AllocatePorts(execution2)
	s.Require().NoError(err)
	s.Len(mappings2, 2)
	s.Equal(1500, mappings2[0].Static)
	s.Equal(dynamicPort, mappings2[1].Static)
}

func (s *PortAllocatorTestSuite) TestExhaustedPortRange() {
	// Create allocator with very small range
	pa, err := compute.NewPortAllocator(3000, 3001) // Only 2 ports available
	s.Require().NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Target: 80},   // Will take first port
			{Target: 8080}, // Will take second port
			{Target: 9090}, // Should fail - no ports left
		},
	}

	_, err = pa.AllocatePorts(execution)
	s.Error(err)
	s.Contains(err.Error(), "no available ports in range")
}

func (s *PortAllocatorTestSuite) TestTargetPortDefaultsToStatic() {
	pa, err := compute.NewPortAllocator(3000, 4000)
	s.Require().NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Network = &models.NetworkConfig{
		Type: models.NetworkHost,
		Ports: []*models.PortMapping{
			{Static: 1500}, // No target port specified
		},
	}

	mappings, err := pa.AllocatePorts(execution)
	s.Require().NoError(err)
	s.Len(mappings, 1)
	s.Equal(1500, mappings[0].Static)
	s.Equal(mappings[0].Static, mappings[0].Target) // Target should match static
}
