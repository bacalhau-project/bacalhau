package models

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExecutionTestSuite struct {
	suite.Suite
}

func TestExecutionSuite(t *testing.T) {
	suite.Run(t, new(ExecutionTestSuite))
}

func (s *ExecutionTestSuite) TestAllocatePorts() {
	tests := []struct {
		name         string
		networkType  Network
		initialPorts []*PortMapping
		allocated    []PortMapping
		want         []*PortMapping
	}{
		{
			name:        "allocate host mode ports",
			networkType: NetworkHost,
			initialPorts: []*PortMapping{
				{
					Name: "http",
					// No static port - should be allocated
				},
			},
			allocated: []PortMapping{
				{
					Name:   "http",
					Static: 8080, // Allocated port
				},
			},
			want: []*PortMapping{
				{
					Name:   "http",
					Static: 8080,
				},
			},
		},
		{
			name:        "allocate bridge mode ports",
			networkType: NetworkBridge,
			initialPorts: []*PortMapping{
				{
					Name:   "http",
					Target: 80,
					// No static port - should be allocated
				},
			},
			allocated: []PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
			},
			want: []*PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
			},
		},
		{
			name:        "multiple port allocations",
			networkType: NetworkBridge,
			initialPorts: []*PortMapping{
				{
					Name:   "http",
					Target: 80,
				},
				{
					Name:   "https",
					Target: 443,
				},
			},
			allocated: []PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
				{
					Name:   "https",
					Static: 8443,
					Target: 443,
				},
			},
			want: []*PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
				{
					Name:   "https",
					Static: 8443,
					Target: 443,
				},
			},
		},
		{
			name:        "preserve host network binding",
			networkType: NetworkBridge,
			initialPorts: []*PortMapping{
				{
					Name:        "http",
					Target:      80,
					HostNetwork: "127.0.0.1",
				},
			},
			allocated: []PortMapping{
				{
					Name:        "http",
					Static:      8080,
					Target:      80,
					HostNetwork: "127.0.0.1",
				},
			},
			want: []*PortMapping{
				{
					Name:        "http",
					Static:      8080,
					Target:      80,
					HostNetwork: "127.0.0.1",
				},
			},
		},
		{
			name:         "nil network config",
			networkType:  NetworkNone,
			initialPorts: nil,
			allocated:    nil,
			want:         []*PortMapping{},
		},
		{
			name:         "empty port list",
			networkType:  NetworkBridge,
			initialPorts: []*PortMapping{},
			allocated:    []PortMapping{},
			want:         []*PortMapping{},
		},
		{
			name:        "replace existing static ports",
			networkType: NetworkBridge,
			initialPorts: []*PortMapping{
				{
					Name:   "http",
					Static: 9090,
					Target: 80,
				},
			},
			allocated: []PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
			},
			want: []*PortMapping{
				{
					Name:   "http",
					Static: 8080,
					Target: 80,
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			execution := &Execution{
				Job: &Job{
					Tasks: []*Task{
						{
							Network: &NetworkConfig{
								Type:  tt.networkType,
								Ports: tt.initialPorts,
							},
						},
					},
				},
			}

			execution.AllocatePorts(tt.allocated)

			if tt.want == nil {
				s.Nil(execution.Job.Task().Network.Ports)
			} else {
				s.Equal(tt.want, execution.Job.Task().Network.Ports)

				// Verify we have a deep copy
				if len(tt.allocated) > 0 {
					tt.allocated[0].Static = 9090
					s.NotEqual(9090, execution.Job.Task().Network.Ports[0].Static)
				}
			}
		})
	}
}
