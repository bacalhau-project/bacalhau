//go:build !integration

package targzip

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/require"
)

const (
	testModeFile = "file"
	testModeDir  = "dir"
	testModePwd  = "pwd"
)

type errorChecker func(require.TestingT, error, ...interface{})

var testSizes = map[datasize.ByteSize]errorChecker{
	0 * datasize.B:                    require.NoError,
	1 * datasize.B:                    require.NoError,
	MaximumContextSize:                require.NoError,
	MaximumContextSize + 1*datasize.B: require.Error,
}

func setup(t *testing.T, size datasize.ByteSize, mode string) (tgzFile *os.File, tgzInput string) {
	rootDir, err := os.MkdirTemp(os.TempDir(), "bacalhau-targzip-test*")
	require.NoError(t, err)

	testDir := rootDir
	if mode == testModeDir {
		testDir = filepath.Join(testDir, "testdir")
		err := os.Mkdir(testDir, worldReadOwnerWritePermission)
		require.NoError(t, err)
	}

	testFilename := filepath.Join(testDir, "data.txt")
	testFile, err := os.Create(testFilename)
	require.NoError(t, err)

	err = testFile.Truncate(int64(size))
	require.NoError(t, err)
	testFile.Close()

	scratchDir, err := os.MkdirTemp(os.TempDir(), "bacalhau-targzip-test*")
	require.NoError(t, err)
	tgzFilename := filepath.Join(scratchDir, "data.tgz")
	tgzFile, err = os.Create(tgzFilename)
	require.NoError(t, err)

	err = os.Chdir(rootDir)
	require.NoError(t, err)
	relFilename, err := filepath.Rel(rootDir, testFilename)
	require.NoError(t, err)
	relDir, err := filepath.Rel(rootDir, testDir)
	require.NoError(t, err)

	tgzInput = map[string]string{
		testModeFile: relFilename,
		testModeDir:  relDir,
		testModePwd:  ".",
	}[mode]
	return tgzFile, tgzInput
}

func TestRoundTrip(t *testing.T) {
	for mode, expectedFile := range map[string]string{
		testModeFile: "data.txt",
		testModeDir:  filepath.Join("testdir", "data.txt"),
		testModePwd:  "data.txt",
	} {
		t.Run(mode, func(t *testing.T) {
			tgzFile, tgzInput := setup(t, datasize.KB, mode)
			defer tgzFile.Close()

			err := Compress(context.Background(), tgzInput, tgzFile)
			require.NoError(t, err)

			_, err = tgzFile.Seek(0, 0)
			require.NoError(t, err)

			outputDir := filepath.Join(t.TempDir(), "outdir")
			err = Decompress(tgzFile, outputDir)
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(outputDir, expectedFile))
		})
	}
}

func TestCompressionSizeLimiting(t *testing.T) {
	for _, mode := range []string{testModeFile, testModeDir} {
		for size, errorChecker := range testSizes {
			t.Run(mode+"/"+size.String(), func(t *testing.T) {
				tgzFile, tgzInput := setup(t, size, mode)
				defer tgzFile.Close()

				err := Compress(context.Background(), tgzInput, tgzFile)
				errorChecker(t, err)
			})
		}
	}
}

func TestDecompressionSizeLimiting(t *testing.T) {
	for _, mode := range []string{testModeFile, testModeDir} {
		for size, errorChecker := range testSizes {
			t.Run(mode+"/"+size.String(), func(t *testing.T) {
				tgzFile, tgzInput := setup(t, size, mode)
				defer tgzFile.Close()

				err := compress(context.Background(), tgzInput, tgzFile, size*2)
				require.NoError(t, err)

				_, err = tgzFile.Seek(0, 0)
				require.NoError(t, err)

				outputDir := filepath.Join(t.TempDir(), "outdir")
				err = Decompress(tgzFile, outputDir)
				errorChecker(t, err)
			})
		}
	}
}
