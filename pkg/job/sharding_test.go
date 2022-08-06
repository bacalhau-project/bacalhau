package job

import (
	"fmt"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestJobShardingSuite(t *testing.T) {
	suite.Run(t, new(JobShardingSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type JobShardingSuite struct {
	suite.Suite
}

// Before all suite
func (suite *JobShardingSuite) SetupAllSuite() {

}

// Before each test
func (suite *JobShardingSuite) SetupTest() {
}

func (suite *JobShardingSuite) TearDownTest() {
}

func (suite *JobShardingSuite) TearDownAllSuite() {

}

func explodeStringArray(arr []string) []storage.StorageSpec {
	results := []storage.StorageSpec{}
	for _, str := range arr {
		results = append(results, storage.StorageSpec{
			Engine: storage.StorageSourceIPFS,
			Path:   str,
		})
	}
	return results
}

func joinStringArray(arr []storage.StorageSpec) []string {
	results := []string{}
	for _, str := range arr {
		results = append(results, str.Path)
	}
	return results
}

func (suite *JobShardingSuite) TestApplyGlobPattern() {

	simpleFileList := []string{
		"/a",
		"/a/file1.txt",
		"/a/file2.txt",
		"/b",
		"/b/file1.txt",
		"/b/file2.txt",
	}

	testCases := []struct {
		name     string
		files    []string
		pattern  string
		basePath string
		outputs  []string
	}{
		{
			"top level folders",
			simpleFileList,
			"/*",
			"",
			[]string{"/a", "/b"},
		},
		{
			"everything",
			simpleFileList,
			"/**/*",
			"",
			simpleFileList,
		},
		{
			"only files in folders",
			simpleFileList,
			"/**/*.*",
			"",
			[]string{
				"/a/file1.txt",
				"/a/file2.txt",
				"/b/file1.txt",
				"/b/file2.txt",
			},
		},
		{
			"base path",
			[]string{
				"/a",
				"/a/file1.txt",
				"/a/file2.txt",
				"/a/file3.txt",
				"/a/file4.txt",
				"/a/apples.txt",
			},
			"/file*.txt",
			"/a",
			[]string{
				"/a/file1.txt",
				"/a/file2.txt",
				"/a/file3.txt",
				"/a/file4.txt",
			},
		},
	}

	for _, testCase := range testCases {
		results, err := ApplyGlobPattern(explodeStringArray(testCase.files), testCase.pattern, testCase.basePath)
		require.NoError(suite.T(), err)
		require.Equal(
			suite.T(),
			strings.Join(testCase.outputs, ","),
			strings.Join(joinStringArray(results), ","),
			fmt.Sprintf("%s: %s did not result in correct answer", testCase.name, testCase.pattern),
		)
	}

}
