package executor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"go.uber.org/multierr"
)

type outputResult struct {
	contents     io.Reader
	filename     string
	fileLimit    int
	summary      *string
	summaryLimit int
	truncated    *bool
}

func writeOutputResult(resultsDir string, output outputResult) error {
	var err error

	// Consume the passed buffers up to the limit of the maximum bytes. The
	// buffers will then contain whatever is left that overflows, and we can
	// write that directly to disk rather than needing to hold it all in memory.
	summary := make([]byte, output.summaryLimit+1)
	summaryRead, err := output.contents.Read(summary)

	available := system.Min(summaryRead, output.summaryLimit)

	if output.summary != nil {
		*(output.summary) = string(summary[:available])
	}
	if output.truncated != nil {
		*(output.truncated) = summaryRead > output.summaryLimit
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
	available = system.Min(summaryRead, output.fileLimit)
	fileWritten, err := file.Write(summary[:available])
	if err != nil && err != io.EOF {
		return err
	}

	_, err = io.CopyN(file, output.contents, int64(output.fileLimit-fileWritten))
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
			system.MaxStdoutFileLengthInBytes,
			&result.STDOUT,
			system.MaxStdoutReturnLengthInBytes,
			&result.StdoutTruncated,
		},
		// Standard error
		{
			stderr,
			ipfs.DownloadFilenameStderr,
			system.MaxStderrFileLengthInBytes,
			&result.STDERR,
			system.MaxStderrReturnLengthInBytes,
			&result.StderrTruncated,
		},
		// Exit code
		{
			bytes.NewReader([]byte(fmt.Sprint(exitcode))),
			ipfs.DownloadFilenameExitCode,
			4,
			nil,
			4,
			nil,
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(outputs))

	errChan := make(chan error, len(outputs))
	makeResult := func(output outputResult) {
		errChan <- writeOutputResult(resultsDir, output)
		wg.Done()
	}

	for _, output := range outputs {
		go makeResult(output)
	}
	wg.Wait()
	close(errChan)

	for outuptErr := range errChan {
		err = multierr.Append(err, outuptErr)
	}

	result.ExitCode = exitcode
	if err != nil {
		result.ErrorMsg = err.Error()
	}

	return result, err
}
