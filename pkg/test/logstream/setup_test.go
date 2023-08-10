//go:build unit || !integration

package logstream

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type LogStreamTestSuite struct {
	suite.Suite

	ctx   context.Context
	stack *devstack.DevStack
}

func TestLogStreamTestSuite(t *testing.T) {
	suite.Run(t, new(LogStreamTestSuite))
}

func (s *LogStreamTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.stack = testutil.Setup(s.ctx, s.T(), devstack.WithNumberOfHybridNodes(1))
}

func waitForOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool, exec executor.Executor) (io.Reader, error) {
	for i := 0; i < 10; i++ {
		reader, err := exec.GetOutputStream(ctx, executionID, withHistory, follow)
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

func newTestExecution(name string, job model.Job) store.LocalState {
	return *store.NewLocalState(
		uuid.NewString(),
		job,
		name,
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}
