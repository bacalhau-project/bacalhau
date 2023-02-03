//go:build unit || !integration

package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/require"
)

func TestWriteResultContents(t *testing.T) {
	testCases := []string{"test", "", string([]byte{0x00, 0x01})}
	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			var summary string
			var truncated bool
			spec := outputResult{
				contents:     strings.NewReader(testCase),
				filename:     "hello",
				fileLimit:    datasize.ByteSize(len(testCase)),
				summary:      &summary,
				summaryLimit: datasize.ByteSize(len(testCase)),
				truncated:    &truncated,
			}

			resultsDir := t.TempDir()
			err := writeOutputResult(resultsDir, spec)
			require.NoError(t, err)

			require.Equal(t, testCase, summary)
			require.Equal(t, false, truncated)

			contents, err := os.ReadFile(filepath.Join(resultsDir, spec.filename))
			require.NoError(t, err)
			require.Equal(t, testCase, string(contents))
		})
	}
}

func TestWriteResultLimitsEnforced(t *testing.T) {
	for _, testCase := range []struct {
		fileLimit, summaryLimit   datasize.ByteSize
		expectFile, expectSummary string
		expectSummaryTruncated    bool
	}{
		{0, 0, "", "", true},
		{5, 11, "hello", "hello world", false},
		{11, 5, "hello world", "hello", true},
		{100, 100, "hello world", "hello world", false},
	} {
		name := fmt.Sprintf("%d %d", testCase.fileLimit, testCase.summaryLimit)
		t.Run(name, func(t *testing.T) {
			var summary string
			var truncated bool
			spec := outputResult{
				contents:     strings.NewReader("hello world"),
				filename:     "hello",
				fileLimit:    testCase.fileLimit,
				summary:      &summary,
				summaryLimit: testCase.summaryLimit,
				truncated:    &truncated,
			}

			resultsDir := t.TempDir()
			err := writeOutputResult(resultsDir, spec)
			require.NoError(t, err)

			require.Equal(t, testCase.expectSummary, summary)
			require.Equal(t, testCase.expectSummaryTruncated, truncated)

			contents, err := os.ReadFile(filepath.Join(resultsDir, spec.filename))
			require.NoError(t, err)
			require.Equal(t, testCase.expectFile, string(contents))
		})
	}
}

func TestWriteResultHandlesNilPointers(t *testing.T) {
	spec := outputResult{
		contents:     strings.NewReader("hello world"),
		filename:     "whatever",
		fileLimit:    1024,
		summary:      nil,
		summaryLimit: 1024,
		truncated:    nil,
	}

	resultsDir := t.TempDir()
	err := writeOutputResult(resultsDir, spec)
	require.NoError(t, err)
}

func TestJobResult(t *testing.T) {
	tempDir := t.TempDir()
	result, err := WriteJobResults(
		tempDir,
		strings.NewReader("standard output"),
		strings.NewReader("standard error"),
		123,
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, "standard output", result.STDOUT)
	require.Equal(t, false, result.StdoutTruncated)
	require.Equal(t, "standard error", result.STDERR)
	require.Equal(t, false, result.StderrTruncated)
	require.Equal(t, 123, result.ExitCode)
	require.Equal(t, "", result.ErrorMsg)

	for filename, expectedContents := range map[string]string{
		model.DownloadFilenameStdout:   "standard output",
		model.DownloadFilenameStderr:   "standard error",
		model.DownloadFilenameExitCode: "123",
	} {
		actualContents, err := os.ReadFile(filepath.Join(tempDir, filename))
		require.NoError(t, err)
		require.Equal(t, expectedContents, string(actualContents))
	}
}
