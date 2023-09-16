package rawexec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type executionHandler struct {
	//
	// provided by the executor
	logger zerolog.Logger

	// the command
	cmd exec.Cmd

	//
	// meta data about the task
	executionID string
	resultsDir  string
	limits      executor.OutputLimits

	//
	// synchronization
	// blocks until the container starts
	activeCh chan bool
	// blocks until the run method returns
	waitCh chan bool
	// true until the run method returns
	running *atomic.Bool

	//
	// results
	result *models.RunCommandResult
}

func (h *executionHandler) active() bool {
	return h.running.Load()
}

func (h *executionHandler) run(ctx context.Context) {
	h.running.Store(true)
	defer func() {
		if err := h.kill(ctx); err != nil {
			log.Warn().Err(err).Msg("failed to cleanup process")
		}
		h.running.Store(false)
		close(h.waitCh)
	}()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	h.cmd.Stdout = outBuf
	h.cmd.Stderr = errBuf

	if err := h.cmd.Start(); err != nil {
		h.result = &models.RunCommandResult{
			STDOUT:          "",
			StdoutTruncated: false,
			STDERR:          "",
			StderrTruncated: false,
			ExitCode:        1,
			ErrorMsg:        err.Error(),
		}
		return
	}

	if err := h.cmd.Wait(); err != nil {
		// TODO read stderr
		h.result = executor.WriteJobResults(h.resultsDir, outBuf, errBuf, h.cmd.ProcessState.ExitCode(), err, h.limits)
		return
	}
	close(h.activeCh)

	outS := outBuf.String()
	errS := errBuf.String()
	fmt.Println(outS)
	fmt.Println(errS)
	h.result = executor.WriteJobResults(h.resultsDir, outBuf, errBuf, h.cmd.ProcessState.ExitCode(), nil, h.limits)
}

func (h *executionHandler) kill(ctx context.Context) error {
	// TODO this instead
	// h.cmd.Process.Signal()
	if h.cmd.ProcessState != nil {
		return h.cmd.Process.Kill()
	}
	return nil
}
