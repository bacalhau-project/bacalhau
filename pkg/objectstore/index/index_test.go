//go:build unit || !integration

package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IndexTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, &IndexTestSuite{
		ctx: context.Background(),
	})
}

// Helper for tests that will convert the provided strings
// into a list and then return the json encoded []byte
func listToBytes(str []string) []byte {
	bytes, _ := json.Marshal(&str)
	return bytes
}

// Helper for tests that will convert the []byte to a
// list of strings
func bytesToList(data []byte) []string {
	var list []string
	_ = json.Unmarshal([]byte(data), &list)
	return list
}

func (s *IndexTestSuite) TestAddToSet() {
	type testCase struct {
		existingList   []string
		toAdd          string
		expectedResult []string
		expectError    bool
		name           string
	}

	testCases := []testCase{
		{nil, "a", []string{"a"}, false, "no existing list"},
		{[]string{"a", "c"}, "b", []string{"a", "b", "c"}, false, "insert in middle"},
		{[]string{"b", "c"}, "a", []string{"a", "b", "c"}, false, "insert at start"},
		{[]string{"a", "b"}, "c", []string{"a", "b", "c"}, false, "insert at end"},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var input []byte
			if tc.existingList == nil {
				input = nil
			} else {
				input = listToBytes(tc.existingList)
			}

			f := index.AddToSet(tc.toAdd)
			ba, err := f(input)
			if tc.expectError {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
			}

			lst := bytesToList(ba)
			require.Equal(s.T(), tc.expectedResult, lst)
		})
	}
}
