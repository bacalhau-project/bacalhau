package job

import (
	"fmt"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/stretchr/testify/require"
)

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
		results, err := ApplyGlobPattern(explodeStringArray(testCase.files), testCase.pattern)
		require.NoError(t, err)
		require.Equal(
			t,
			strings.Join(testCase.outputs, ","),
			strings.Join(joinStringArray(results), ","),
			fmt.Sprintf("%s: %s did not result in correct answer", testCase.name, testCase.pattern),
		)
	}

}
