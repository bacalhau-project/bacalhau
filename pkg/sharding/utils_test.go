package sharding

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyGlobPattern(t *testing.T) {

	simpleFileList := []string{
		"/a",
		"/a/file1.txt",
		"/a/file2.txt",
		"/b",
		"/b/file1.txt",
		"/b/file2.txt",
	}

	testCases := []struct {
		name    string
		files   []string
		pattern string
		outputs []string
	}{
		{
			"top level folders",
			simpleFileList,
			"/*",
			[]string{"/a", "/b"},
		},
		{
			"everything",
			simpleFileList,
			"/**/*",
			simpleFileList,
		},
		{
			"only files in folders",
			simpleFileList,
			"/**/*.*",
			[]string{
				"/a/file1.txt",
				"/a/file2.txt",
				"/b/file1.txt",
				"/b/file2.txt",
			},
		},
	}

	for _, testCase := range testCases {
		results, err := ApplyGlobPattern(testCase.files, testCase.pattern)
		require.NoError(t, err)
		require.Equal(t, strings.Join(testCase.outputs, ","), strings.Join(results, ","), fmt.Sprintf("%s: %s did not result in correct answer", testCase.name, testCase.pattern))
	}

}
