//go:build unit || !integration

package compute

import (
	"fmt"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

type HealthTrackerTestSuite struct {
	suite.Suite
	clock   *clock.Mock
	tracker *HealthTracker
}

func TestHealthTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(HealthTrackerTestSuite))
}

func (s *HealthTrackerTestSuite) SetupTest() {
	s.clock = clock.NewMock()
	s.tracker = NewHealthTracker(s.clock)
}

func (s *HealthTrackerTestSuite) TestInitialState() {
	startTime := s.clock.Now()
	health := s.tracker.GetHealth()

	s.Require().Equal(nclprotocol.Disconnected, health.CurrentState)
	s.Require().Equal(startTime, health.StartTime)
	s.Require().True(health.LastSuccessfulHeartbeat.IsZero())
	s.Require().True(health.LastSuccessfulUpdate.IsZero())
	s.Require().Equal(0, health.ConsecutiveFailures)
	s.Require().Nil(health.LastError)
	s.Require().True(health.ConnectedSince.IsZero())
}

func (s *HealthTrackerTestSuite) TestMarkConnected() {
	// Advance clock to have distinct timestamps
	s.clock.Add(time.Second)
	connectedTime := s.clock.Now()

	s.tracker.MarkConnected()
	health := s.tracker.GetHealth()

	s.Require().Equal(nclprotocol.Connected, health.CurrentState)
	s.Require().Equal(connectedTime, health.ConnectedSince)
	s.Require().Equal(connectedTime, health.LastSuccessfulHeartbeat)
	s.Require().Equal(0, health.ConsecutiveFailures)
	s.Require().Nil(health.LastError)
}

func (s *HealthTrackerTestSuite) TestMarkDisconnected() {
	// Set up initial connected state
	s.tracker.MarkConnected()

	// Simulate disconnection
	expectedErr := fmt.Errorf("connection lost")
	s.tracker.MarkDisconnected(expectedErr)
	health := s.tracker.GetHealth()

	s.Require().Equal(nclprotocol.Disconnected, health.CurrentState)
	s.Require().Equal(expectedErr, health.LastError)
	s.Require().Equal(1, health.ConsecutiveFailures)

	// Multiple disconnections should increment failure count
	s.tracker.MarkDisconnected(expectedErr)
	health = s.tracker.GetHealth()
	s.Require().Equal(2, health.ConsecutiveFailures)
}

func (s *HealthTrackerTestSuite) TestSuccessfulOperations() {
	// Initial timestamps
	s.clock.Add(time.Second)
	s.tracker.MarkConnected()

	// Test heartbeat success
	s.clock.Add(time.Second)
	heartbeatTime := s.clock.Now()
	s.tracker.HeartbeatSuccess()

	// Test update success
	s.clock.Add(time.Second)
	updateTime := s.clock.Now()
	s.tracker.UpdateSuccess()

	// Verify timestamps
	health := s.tracker.GetHealth()
	s.Require().Equal(heartbeatTime, health.LastSuccessfulHeartbeat)
	s.Require().Equal(updateTime, health.LastSuccessfulUpdate)
}

func (s *HealthTrackerTestSuite) TestConnectionStateTransitions() {
	// Test full connection lifecycle
	states := []struct {
		operation func()
		expected  nclprotocol.ConnectionState
	}{
		{
			operation: func() { s.tracker.MarkConnected() },
			expected:  nclprotocol.Connected,
		},
		{
			operation: func() { s.tracker.MarkDisconnected(fmt.Errorf("error")) },
			expected:  nclprotocol.Disconnected,
		},
		{
			operation: func() { s.tracker.MarkConnected() },
			expected:  nclprotocol.Connected,
		},
	}

	for _, tc := range states {
		tc.operation()
		s.Require().Equal(tc.expected, s.tracker.GetState())
	}
}
