//go:build unit || !integration

package watchers

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type ExecutionTransitionsTestSuite struct {
	suite.Suite
}

func TestExecutionTransitionsTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutionTransitionsTestSuite))
}

func (s *ExecutionTransitionsTestSuite) TestShouldAskForPendingBid() {
	tests := []struct {
		name     string
		previous *models.Execution
		current  *models.Execution
		expected bool
	}{
		{
			name:     "new_pending_execution",
			previous: nil,
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				return e
			}(),
			expected: true,
		},
		{
			name: "existing_pending_execution",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				return e
			}(),
			expected: false,
		},
		{
			name:     "new_running_execution",
			previous: nil,
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			upsert := models.ExecutionUpsert{
				Previous: tt.previous,
				Current:  tt.current,
			}
			transitions := newExecutionTransitions(upsert)
			s.Equal(tt.expected, transitions.shouldAskForPendingBid())
		})
	}
}

func (s *ExecutionTransitionsTestSuite) TestShouldAskForDirectBid() {
	tests := []struct {
		name     string
		previous *models.Execution
		current  *models.Execution
		expected bool
	}{
		{
			name:     "new_running_execution",
			previous: nil,
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: true,
		},
		{
			name: "existing_running_execution",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: false,
		},
		{
			name:     "new_pending_execution",
			previous: nil,
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				return e
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			upsert := models.ExecutionUpsert{
				Previous: tt.previous,
				Current:  tt.current,
			}
			transitions := newExecutionTransitions(upsert)
			s.Equal(tt.expected, transitions.shouldAskForDirectBid())
		})
	}
}

func (s *ExecutionTransitionsTestSuite) TestShouldAcceptBid() {
	tests := []struct {
		name     string
		previous *models.Execution
		current  *models.Execution
		expected bool
	}{
		{
			name: "pending_to_running_transition",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: true,
		},
		{
			name: "running_to_running_transition",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: false,
		},
		{
			name:     "new_running_execution",
			previous: nil,
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			upsert := models.ExecutionUpsert{
				Previous: tt.previous,
				Current:  tt.current,
			}
			transitions := newExecutionTransitions(upsert)
			s.Equal(tt.expected, transitions.shouldAcceptBid())
		})
	}
}

func (s *ExecutionTransitionsTestSuite) TestShouldCancel() {
	tests := []struct {
		name     string
		previous *models.Execution
		current  *models.Execution
		expected bool
	}{
		{
			name: "running_to_stopped_transition",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateRunning)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateRunning)
				return e
			}(),
			expected: true,
		},
		{
			name: "running_to_stopped_already_terminal",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
				return e
			}(),
			expected: false,
		},
		{
			name: "stopped_to_stopped_transition",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				return e
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			upsert := models.ExecutionUpsert{
				Previous: tt.previous,
				Current:  tt.current,
			}
			transitions := newExecutionTransitions(upsert)
			s.Equal(tt.expected, transitions.shouldCancel())
		})
	}
}

func (s *ExecutionTransitionsTestSuite) TestShouldRejectBid() {
	tests := []struct {
		name     string
		previous *models.Execution
		current  *models.Execution
		expected bool
	}{
		{
			name: "pending_to_stopped_with_bid",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateAskForBidAccepted)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateAskForBidAccepted)
				return e
			}(),
			expected: true,
		},
		{
			name: "pending_to_stopped_no_bid",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateNew)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				e.ComputeState = models.NewExecutionState(models.ExecutionStateNew)
				return e
			}(),
			expected: false,
		},
		{
			name: "running_to_stopped_transition",
			previous: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)
				return e
			}(),
			current: func() *models.Execution {
				e := mock.Execution()
				e.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateStopped)
				return e
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			upsert := models.ExecutionUpsert{
				Previous: tt.previous,
				Current:  tt.current,
			}
			transitions := newExecutionTransitions(upsert)
			s.Equal(tt.expected, transitions.shouldRejectBid())
		})
	}
}
