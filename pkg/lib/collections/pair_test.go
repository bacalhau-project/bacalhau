//go:build unit || !integration

package collections

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// PairTestSuite defines the suite for testing Pair.
type PairTestSuite struct {
	suite.Suite
}

func TestPairTestSuite(t *testing.T) {
	suite.Run(t, new(PairTestSuite))
}

// TestNewPair tests the NewPair function for correct Pair creation.
func (suite *PairTestSuite) TestNewPair() {
	pair := NewPair("left", 100)
	suite.Equal("left", pair.Left, "Left value does not match expected value")
	suite.Equal(100, pair.Right, "Right value does not match expected value")
}

// TestPairString tests the String method for the correct string representation of the Pair.
func (suite *PairTestSuite) TestPairString() {
	tests := []struct {
		pair     Pair[any, any]
		expected string
	}{
		{NewPair[any, any]("left", 100), "(left, 100)"},
		{NewPair[any, any](1, 2), "(1, 2)"},
		{NewPair[any, any](0.5, "right"), "(0.5, right)"},
	}

	for _, test := range tests {
		suite.Equal(test.expected, test.pair.String(), "String representation does not match")
	}
}
