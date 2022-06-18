package bacalhau

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

	allBadStrings := LoadBadStringsFull()
	for _, s := range allBadStrings {
		l := struct {
			stringToTest    string
			predictedLength int // set to -1 if skip test
		}{stringToTest: s, predictedLength: -1}
		tests = append(tests, l)
	}

	for _, tc := range tests {
		strippedString := SafeStringStripper(tc.stringToTest)
		assert.LessOrEqual(suite.T(), len(strippedString), len(tc.stringToTest))
		if tc.predictedLength >= 0 {
			assert.Equal(suite.T(), tc.predictedLength, len(strippedString))
		}
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}
