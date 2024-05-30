//go:build unit || !integration

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
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

func (s *UtilsSuite) TestSafeRegex() {
	// Put a few examples at the front, for manual testing
	tests := []struct {
		stringToTest    string
		predictedLength int // set to -1 if skip test
	}{
		{stringToTest: "abc123-", predictedLength: 7},        // Nothing should be stripped
		{stringToTest: `"'@123`, predictedLength: 4},         // Should leave just 123
		{stringToTest: "👫👭👲👴", predictedLength: len("👫👭👲👴")}, // Emojis should work
	}

	for _, tc := range tests {
		strippedString := legacy_job.SafeStringStripper(tc.stringToTest)
		require.LessOrEqual(s.T(), len(strippedString), len(tc.stringToTest))
		if tc.predictedLength >= 0 {
			require.Equal(s.T(), tc.predictedLength, len(strippedString))
		}
	}
}

func (s *UtilsSuite) TestVersionCheck() {
	setup.SetupBacalhauRepoForTesting(s.T())

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

func (s *UtilsSuite) TestImages() {
	tc := map[string]struct {
		Image string
		Valid bool
	}{
		// TODO: #843 Unblock when we can figure out how to check the existence of the image
		// "no image": {
		// 	image: "",
		// 	valid: false,
		// },
		// "invalid image": {
		// 	image: "badimageNOTFOUND",
		// 	valid: false,
		// },
		"image with tag (norepo)": {
			Image: "ubuntu:latest",
			Valid: true,
		},
		"image with tag (repo)": {
			Image: "curlimages/curl:7.85.0",
			Valid: true,
		},
	}

	for name, test := range tc {
		s.Run(name, func() {
			sampleJob, _ := model.NewJobWithSaneProductionDefaults()
			sampleJob.Spec.EngineSpec = model.NewDockerEngineBuilder(test.Image).Build()
			err := legacy_job.VerifyJob(context.TODO(), sampleJob)
			if test.Valid {
				require.NoError(s.T(), err, "%s: expected valid image %s to pass", name, test.Image)
			} else {
				require.Error(s.T(), err, "%s: expected invalid image %s to fail", name, test.Image)
			}
		})
	}
}
