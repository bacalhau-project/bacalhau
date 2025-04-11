package logstream

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type ExecutionLogWriterParams struct {
	logsDir string
}

type ExecutionLogWriter struct {
	params *ExecutionLogWriterParams
}

func NewExecutionLogWriter(logsDir string) (*ExecutionLogWriter, error) {
	return &ExecutionLogWriter{&ExecutionLogWriterParams{logsDir}}, nil
}

func (w *ExecutionLogWriter) StartWriting(src io.Reader) chan util.Result[int64] {
	resultCh := make(chan util.Result[int64])
	go func() {
		defer close(resultCh)
		fileWriter, err := os.Create(filepath.Join(w.params.logsDir, compute.ExecutionLogFileName))
		if err != nil {
			resultCh <- util.NewResult[int64](0, err)
			return
		}
		defer closer.CloseWithLogOnError("executionLogWriter", fileWriter)

		copiedBytes, err := StdCopyWithEndFrame(fileWriter, src)
		// Write the result to the channel, but don't block if the channel is full or nothing is reading.
		select {
		case resultCh <- util.NewResult(copiedBytes, err):
		default:
		}
	}()
	return resultCh
}
