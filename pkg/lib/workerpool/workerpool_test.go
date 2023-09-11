//go:build unit || !integration

package workerpool_test

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/workerpool"
	"github.com/stretchr/testify/suite"
)

type WorkerPoolSuite struct {
	suite.Suite
}

func TestWorkerPoolSuite(t *testing.T) {
	suite.Run(t, new(WorkerPoolSuite))
}

func (s *WorkerPoolSuite) TestStartAndCancel() {
	ctx, cancel := context.WithCancel(context.Background())

	w, err := workerpool.NewWorkerPool[string](
		func(m string) error {
			return nil
		},
		workerpool.WithWorkerCount(2),
	)
	s.Require().NoError(err)

	w.Start(ctx)
	cancel()

	err = w.Shutdown(10 * time.Millisecond)
	s.Require().NoError(err)
}

func (s *WorkerPoolSuite) TestBusy() {
	ctr := 0
	start := time.Now()

	number := 1000
	ctx := context.Background()
	w, err := workerpool.NewWorkerPool[string](
		func(m string) error {
			ctr += 1
			return nil
		},
		workerpool.WithWorkerCount(50),
		workerpool.WithInputChannelSize(16),
	)
	s.Require().NoError(err)

	w.Start(ctx)

	for i := 0; i < number; i++ {
		w.Submit("a msg")
	}

	err = w.Shutdown(10 * time.Millisecond)
	took := time.Since(start)

	// We expect no error, and a fast processing time but between the submit and
	// the shutdown we may not manage to process every single message.
	s.Require().NoError(err)
	s.Require().Less(took, time.Duration(50*time.Millisecond))
}
