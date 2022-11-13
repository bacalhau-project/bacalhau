//go:build unit || !integration

package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContents(t *testing.T) {
	testCases := []string{"test", "", string([]byte{0x00, 0x01})}
	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			var summary string
			var truncated bool
			spec := outputResult{
				contents:     strings.NewReader(testCase),
				filename:     "hello",
				fileLimit:    len(testCase),
				summary:      &summary,
				summaryLimit: len(testCase),
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

func TestLimitsEnforced(t *testing.T) {
	for _, testCase := range []struct {
		fileLimit, summaryLimit   int
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

func TestHandlesNilPointers(t *testing.T) {
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
