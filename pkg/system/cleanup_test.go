//go:build unit || !integration

package system

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemCleanupSuite struct {
	suite.Suite
}

func TestSystemCleanupSuite(t *testing.T) {
	suite.Run(t, new(SystemCleanupSuite))
}

func (s *SystemCleanupSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	require.NoError(s.T(), InitConfigForTesting(s.T()))
}

func (s *SystemCleanupSuite) TestCleanupManager() {
	clean := false
	cleanWithContext := false

	cm := NewCleanupManager()

	cm.RegisterCallback(func() error {
		clean = true
		return nil
	})
	cm.RegisterCallbackWithContext(func(ctx context.Context) error {
		s.NoError(ctx.Err())
		cleanWithContext = true
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cm.Cleanup(ctx)
	s.True(clean, "cleanup handler failed to run registered functions")
	s.True(cleanWithContext)
}
