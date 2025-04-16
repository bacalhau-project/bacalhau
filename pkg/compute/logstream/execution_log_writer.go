package logstream

import (
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
)

type ExecutionLogWriterParams struct {
	logsDir string
}

type ExecutionLogWriter struct {
	isClosed   bool
	fileHandle *os.File
	params     *ExecutionLogWriterParams
}

func NewExecutionLogWriter(logsDir string) (*ExecutionLogWriter, error) {
	return &ExecutionLogWriter{
		params: &ExecutionLogWriterParams{logsDir},
	}, nil
}

func (cw *ExecutionLogWriter) Write(p []byte) (int, error) {
	fileHandle, err := cw.getFileHandle()

	if err != nil {
		return 0, err
	}

	return fileHandle.Write(p)
}

func (cw *ExecutionLogWriter) Close() error {
	if cw.isClosed {
		return nil
	}
	cw.isClosed = true
	// Close the file if it was opened
	if cw.fileHandle != nil {
		if err := cw.fileHandle.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (cw *ExecutionLogWriter) getFileHandle() (*os.File, error) {
	if cw.fileHandle != nil {
		return cw.fileHandle, nil
	}
	var err error
	cw.fileHandle, err = os.Create(filepath.Join(cw.params.logsDir, compute.ExecutionLogFileName))
	if err != nil {
		return nil, err
	}
	return cw.fileHandle, nil
}
