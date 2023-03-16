//go:build unit || !integration

package logstream

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type LogStreamTestSuite struct {
	suite.Suite

	ctx   context.Context
	stack *devstack.DevStack
	cm    *system.CleanupManager
}

func TestLogStreamTestSuite(t *testing.T) {
	suite.Run(t, new(LogStreamTestSuite))
}

func (s *LogStreamTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.stack, s.cm = testutil.SetupTestWithDefaultConfigs(s.ctx, s.T(), 1, 0, false)
}

func (s *LogStreamTestSuite) TearDownSuite() {
	s.cm.Cleanup(s.ctx)
}

func waitForOutputStream(ctx context.Context, job model.Job, withHistory bool, exec executor.Executor) (io.Reader, error) {
	for i := 0; i < 10; i++ {
		reader, err := exec.GetOutputStream(ctx, job, withHistory)
		if err != nil {
			if strings.Contains(err.Error(), "not implemented") {
				return nil, err
			}

			time.Sleep(time.Duration(500) * time.Millisecond)
		}
		if reader != nil {
			return reader, nil
		}
	}

	return nil, fmt.Errorf("failed to get output stream from container")
}

func newTestExecution(name string, job model.Job) store.Execution {
	return *store.NewExecution(
		uuid.NewString(),
		job,
		name,
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}

func newWasmJob(id string, spec model.JobSpecWasm) model.Job {
	return model.Job{
		Metadata: model.Metadata{
			ID: id,
		},
		Spec: model.Spec{
			Engine: model.EngineWasm,
			Wasm:   spec,
		},
	}
}

func newDockerJob(id string, spec model.JobSpecDocker) model.Job {
	return model.Job{
		Metadata: model.Metadata{
			ID: id,
		},
		Spec: model.Spec{
			Engine: model.EngineDocker,
			Docker: spec,
		},
	}

}
