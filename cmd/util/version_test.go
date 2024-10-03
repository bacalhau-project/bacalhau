//go:build unit || !integration

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type UtilsSuite struct {
	suite.Suite
}

// Before each test
func (s *UtilsSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *UtilsSuite) TestVersionCheck() {
	// OK: Normal operation
	err := EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "v1.2.3",
	}, &models.BuildVersionInfo{
		GitVersion: "v1.2.3",
	})
	require.NoError(s.T(), err)

	// OK: invalid semver
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "not-a-sem-ver",
	}, &models.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: nil semver
	err = EnsureValidVersion(context.TODO(), nil, &models.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: development version
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: version.DevelopmentGitVersion,
	}, &models.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: development version
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "v1.2.0",
	}, &models.BuildVersionInfo{
		GitVersion: version.DevelopmentGitVersion,
	})
	require.NoError(s.T(), err)

	// NOT OK: server is newer
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "v1.2.3",
	}, &models.BuildVersionInfo{
		GitVersion: "v1.2.4",
	})
	require.Error(s.T(), err)

	// NOT OK: client is newer
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "v1.2.4",
	}, &models.BuildVersionInfo{
		GitVersion: "v1.2.3",
	})
	require.Error(s.T(), err)

	// https://github.com/bacalhau-project/bacalhau/issues/495
	err = EnsureValidVersion(context.TODO(), &models.BuildVersionInfo{
		GitVersion: "v0.1.37",
	}, &models.BuildVersionInfo{
		GitVersion: "v0.1.36",
	})
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "client version v0.1.37")
}
