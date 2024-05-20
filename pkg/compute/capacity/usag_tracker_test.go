//go:build unit || !integration

package capacity

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type LocalUsageTrackerTestSuite struct {
	suite.Suite
	tracker *LocalUsageTracker
}

func (s *LocalUsageTrackerTestSuite) SetupTest() {
	s.tracker = NewLocalUsageTracker()
}

func (s *LocalUsageTrackerTestSuite) TestAddAndRemove() {
	ctx := context.Background()
	usage := models.Resources{CPU: 2, Memory: 1024, Disk: 10000, GPU: 1}

	// Test Add
	s.tracker.Add(ctx, usage)

	usedCapacity := s.tracker.GetUsedCapacity(ctx)
	s.Require().Equal(usage, usedCapacity)

	// Test Remove
	s.tracker.Remove(ctx, usage)

	usedCapacity = s.tracker.GetUsedCapacity(ctx)
	s.Require().True(usedCapacity.IsZero())
}

func TestLocalUsageTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(LocalUsageTrackerTestSuite))
}
