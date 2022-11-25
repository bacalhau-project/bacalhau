package executor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.ptx.dk/multierrgroup"
	"go.uber.org/multierr"
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
	var err error

	// Consume the passed buffers up to the limit of the maximum bytes. The
	// buffers will then contain whatever is left that overflows, and we can
	// write that directly to disk rather than needing to hold it all in memory.
	summary := make([]byte, output.summaryLimit+1)
	summaryRead, err := output.contents.Read(summary)

	available := system.Min(summaryRead, int(output.summaryLimit))

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
	defer file.Close()

	// First write the bytes we have already read, and then write whatever
	// is left in the buffer, but only up to the maximum file limit.
	available = system.Min(summaryRead, int(output.fileLimit))
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

// WriteJobResult produces files and a model.RunCommandResult in the standard
// format, including truncating the contents of both where necessary to fit
// within system-defined limits.
//
// It will consume only the bytes from the passed io.Readers that it needs to
// correctly form job outputs. Once the command returns, the readers can close.
func WriteJobResults(resultsDir string, stdout, stderr io.Reader, exitcode int, err error) (*model.RunCommandResult, error) {
	result := model.NewRunCommandResult()

	outputs := []outputResult{
		// Standard output
		{
			stdout,
			ipfs.DownloadFilenameStdout,
			system.MaxStdoutFileLength,
			&result.STDOUT,
			system.MaxStdoutReturnLength,
			&result.StdoutTruncated,
		},
		// Standard error
		{
			stderr,
			ipfs.DownloadFilenameStderr,
			system.MaxStderrFileLength,
			&result.STDERR,
			system.MaxStderrReturnLength,
			&result.StderrTruncated,
		},
		// Exit code
		{
			strings.NewReader(fmt.Sprint(exitcode)),
			ipfs.DownloadFilenameExitCode,
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
