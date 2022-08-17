package bacalhau

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type UtilsSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *UtilsSuite) SetupAllSuite() {

}

// Before each test
func (suite *UtilsSuite) SetupTest() {
	suite.rootCmd = RootCmd
}

func (suite *UtilsSuite) TearDownTest() {
}

func (suite *UtilsSuite) TearDownAllSuite() {

}

func (suite *UtilsSuite) TestSafeRegex() {
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
		require.LessOrEqual(suite.T(), len(strippedString), len(tc.stringToTest))
		if tc.predictedLength >= 0 {
			require.Equal(suite.T(), tc.predictedLength, len(strippedString))
		}
	}
}

func (suite *UtilsSuite) TestVersionCheck() {
	system.InitConfigForTesting(suite.T())

	// OK: Normal operation
	err := ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "v1.2.3",
	}, &executor.VersionInfo{
		GitVersion: "v1.2.3",
	})
	require.NoError(suite.T(), err)

	// OK: invalid semver
	err = ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "not-a-sem-ver",
	}, &executor.VersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(suite.T(), err)

	// OK: nil semver
	err = ensureValidVersion(context.TODO(), nil, &executor.VersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(suite.T(), err)

	// OK: development version
	err = ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "v0.0.0-xxxxxxx",
	}, &executor.VersionInfo{
		GitVersion: "v1.2.0",
	})
	require.NoError(suite.T(), err)

	// NOT OK: server is newer
	err = ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "v1.2.3",
	}, &executor.VersionInfo{
		GitVersion: "v1.2.4",
	})
	require.Error(suite.T(), err)

	// NOT OK: client is newer
	err = ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "v1.2.4",
	}, &executor.VersionInfo{
		GitVersion: "v1.2.3",
	})
	require.Error(suite.T(), err)

	// https://github.com/filecoin-project/bacalhau/issues/495
	err = ensureValidVersion(context.TODO(), &executor.VersionInfo{
		GitVersion: "v0.1.37",
	}, &executor.VersionInfo{
		GitVersion: "v0.0.0-xxxxxxx",
	})
	require.Error(suite.T(), err)
	require.Contains(suite.T(), err.Error(), "client version v0.1.37")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
