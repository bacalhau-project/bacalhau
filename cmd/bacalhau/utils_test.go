//go:build !integration

package bacalhau

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type UtilsSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before each test
func (s *UtilsSuite) SetupTest() {
	s.rootCmd = RootCmd
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
		strippedString := job.SafeStringStripper(tc.stringToTest)
		require.LessOrEqual(s.T(), len(strippedString), len(tc.stringToTest))
		if tc.predictedLength >= 0 {
			require.Equal(s.T(), tc.predictedLength, len(strippedString))
		}
	}
}

func (s *UtilsSuite) TestVersionCheck() {
	require.NoError(s.T(), system.InitConfigForTesting())

	// OK: Normal operation
	err := ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v1.2.3",
	}, &model.BuildVersionInfo{
		GitVersion: "v1.2.3",
	})
	require.NoError(s.T(), err)

	// OK: invalid semver
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "not-a-sem-ver",
	}, &model.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: nil semver
	err = ensureValidVersion(context.TODO(), nil, &model.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: development version
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v0.0.0-xxxxxxx",
	}, &model.BuildVersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(s.T(), err)

	// OK: development version
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v1.2.0",
	}, &model.BuildVersionInfo{
		GitVersion: "v0.0.0-xxxxxxx",
	})
	require.NoError(s.T(), err)

	// NOT OK: server is newer
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v1.2.3",
	}, &model.BuildVersionInfo{
		GitVersion: "v1.2.4",
	})
	require.Error(s.T(), err)

	// NOT OK: client is newer
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v1.2.4",
	}, &model.BuildVersionInfo{
		GitVersion: "v1.2.3",
	})
	require.Error(s.T(), err)

	// https://github.com/filecoin-project/bacalhau/issues/495
	err = ensureValidVersion(context.TODO(), &model.BuildVersionInfo{
		GitVersion: "v0.1.37",
	}, &model.BuildVersionInfo{
		GitVersion: "v0.1.36",
	})
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "client version v0.1.37")
}

func (s *UtilsSuite) TestImages() {
	tc := map[string]struct {
		image string
		valid bool
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
			image: "ubuntu:latest",
			valid: true,
		},
		"image with tag (repo)": {
			image: "curlimages/curl:7.85.0",
			valid: true,
		},
	}

	for name, test := range tc {
		s.Run(name, func() {
			sampleJob, _ := model.NewJobWithSaneProductionDefaults()
			sampleJob.Spec.Docker.Image = test.image
			err := job.VerifyJob(context.TODO(), sampleJob)
			if test.valid {
				require.NoError(s.T(), err, "%s: expected valid image %s to pass", name, test.image)
			} else {
				require.Error(s.T(), err, "%s: expected invalid image %s to fail", name, test.image)
			}
		})
	}
}
