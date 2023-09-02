package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/c2h5oh/datasize"
	"go.ptx.dk/multierrgroup"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type outputResult struct {
	contents     io.Reader
	filename     string
	fileLimit    datasize.ByteSize
	summary      *string
	summaryLimit datasize.ByteSize
	truncated    *bool
}

func writeOutputResult(resultsDir string, output outputResult) error {
	if output.contents == nil {
		// contents may be nil if something went wrong while trying to get the logs
		output.contents = bytes.NewReader(nil)
	}

	var err error

	// Consume the passed buffers up to the limit of the maximum bytes. The
	// buffers will then contain whatever is left that overflows, and we can
	// write that directly to disk rather than needing to hold it all in memory.
	summary := make([]byte, output.summaryLimit+1)
	summaryRead, err := output.contents.Read(summary)
	if err != nil && err != io.EOF {
		return err
	}

	available := math.Min(summaryRead, int(output.summaryLimit))

	if output.summary != nil {
		*(output.summary) = string(summary[:available])
	}
	if output.truncated != nil {
		*(output.truncated) = summaryRead > int(output.summaryLimit)
	}
	if err != nil && err != io.EOF {
		return err
	}

	file, err := os.Create(filepath.Join(resultsDir, output.filename))
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError("file", file)

	// First write the bytes we have already read, and then write whatever
	// is left in the buffer, but only up to the maximum file limit.
	available = math.Min(summaryRead, int(output.fileLimit))
	fileWritten, err := file.Write(summary[:available])
	if err != nil && err != io.EOF {
		return err
	}

	_, err = io.CopyN(file, output.contents, int64(int(output.fileLimit)-fileWritten))
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

type OutputLimits struct {
	MaxStdoutFileLength   datasize.ByteSize
	MaxStdoutReturnLength datasize.ByteSize
	MaxStderrFileLength   datasize.ByteSize
	MaxStderrReturnLength datasize.ByteSize
}

// WriteJobResults produces files and a models.RunCommandResult in the standard
// format, including truncating the contents of both where necessary to fit
// within system-defined limits.
//
// It will consume only the bytes from the passed io.Readers that it needs to
// correctly form job outputs. Once the command returns, the readers can close.
func WriteJobResults(resultsDir string, stdout, stderr io.Reader,
	exitcode int,
	err error,
	limits OutputLimits) (*models.RunCommandResult, error) {
	result := models.NewRunCommandResult()

	outputs := []outputResult{
		// Standard output
		{
			stdout,
			models.DownloadFilenameStdout,
			limits.MaxStdoutFileLength,
			&result.STDOUT,
			limits.MaxStdoutReturnLength,
			&result.StdoutTruncated,
		},
		// Standard error
		{
			stderr,
			models.DownloadFilenameStderr,
			limits.MaxStderrFileLength,
			&result.STDERR,
			limits.MaxStderrReturnLength,
			&result.StderrTruncated,
		},
		// Exit code
		{
			strings.NewReader(fmt.Sprint(exitcode)),
			models.DownloadFilenameExitCode,
			4,
			nil,
			4,
			nil,
		},
	}

	wg := multierrgroup.Group{}
	for _, output := range outputs {
		output := output
		wg.Go(func() error {
			return writeOutputResult(resultsDir, output)
		})
	}

	err = multierr.Append(err, wg.Wait())
	if err != nil {
		result.ErrorMsg = err.Error()
	}

	result.ExitCode = exitcode
	return result, err
}

func FailResult(err error) (*models.RunCommandResult, error) {
	return &models.RunCommandResult{ErrorMsg: err.Error()}, err
}
