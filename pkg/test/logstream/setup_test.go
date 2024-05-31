//go:build unit || !integration

package logstream_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/setup"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
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
	fsr, c := setup.SetupBacalhauRepoForTesting(s.T())
	s.stack = testutil.Setup(s.ctx, s.T(), fsr, c, devstack.WithNumberOfHybridNodes(1))
}

func waitForOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool, exec executor.Executor) (chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	for i := 0; i < 10; i++ {
		reader, err := exec.GetLogStream(ctx, executor.LogStreamRequest{
			ExecutionID: executionID,
			Tail:        withHistory,
			Follow:      follow,
		})
		if err != nil {
			if strings.Contains(err.Error(), "not implemented") {
				return nil, err
			}

			time.Sleep(time.Duration(500) * time.Millisecond)
		}
		if reader != nil {
			streamer := logstream.NewLiveStreamer(logstream.LiveStreamerParams{
				Reader: reader,
			})
			ch := streamer.Stream(ctx)
			return ch, nil
		}
	}

	return nil, fmt.Errorf("failed to get output stream from container")
}
