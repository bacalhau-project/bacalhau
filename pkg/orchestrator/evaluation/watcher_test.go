//go:build unit || !integration

package evaluation_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type WatcherTestSuite struct {
	suite.Suite
	store   jobstore.Store
	broker  *evaluation.InMemoryBroker
	watcher *evaluation.Watcher
}

func (s *WatcherTestSuite) SetupTest() {
	// Open the Bolt database on the temporary file.
	var err error
	s.store, err = boltjobstore.NewBoltJobStore(filepath.Join(s.T().TempDir(), "evaluation-watcher.db"))
	s.Require().NoError(err)

	s.broker, err = evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{})
	s.Require().NoError(err)
	s.broker.SetEnabled(true)

	s.watcher = evaluation.NewWatcher(s.store, s.broker)
}

func (s *WatcherTestSuite) TearDownTest() {
	_ = s.store.Close(context.Background())
}

func (s *WatcherTestSuite) TestEnqueueEval() {
	// start the watcher
	s.watcher.Start(context.Background())
	s.Eventually(func() bool {
		return s.watcher.IsWatching()
	}, 100*time.Millisecond, 10*time.Millisecond)

	eval, err := s.createEvaluation()

	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 100*time.Millisecond)
	s.Require().NoError(err)
	s.Require().NotNil(dequeuedEval, "evaluation should be enqueued")
	s.Require().Equal(eval.ID, dequeuedEval.ID)

	// no more evaluations
	dequeuedEval, _, err = s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "evaluation should not be enqueued")
}

// TestStoppedWatcher tests that the watcher stops watching when the context is canceled
func (s *WatcherTestSuite) TestStoppedWatcher_ContextDone() {
	// start the watcher
	ctx, cancel := context.WithCancel(context.Background())
	s.watcher.Start(ctx)
	s.Eventually(func() bool {
		return s.watcher.IsWatching()
	}, 100*time.Millisecond, 10*time.Millisecond)

	cancel()
	s.Eventually(func() bool {
		return !s.watcher.IsWatching()
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Create an evaluation
	eval, err := s.createEvaluation()
	s.Require().NoError(err)

	// no evaluations should be enqueued
	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "evaluation should not be enqueued")
}

// TestStoppedWatcher_StopCalled tests that the watcher stops watching when Stop is called
func (s *WatcherTestSuite) TestStoppedWatcher_StopCalled() {
	// start the watcher
	s.watcher.Start(context.Background())
	s.Eventually(func() bool {
		return s.watcher.IsWatching()
	}, 100*time.Millisecond, 10*time.Millisecond)

	s.watcher.Stop()
	s.Eventually(func() bool {
		return !s.watcher.IsWatching()
	}, 100*time.Millisecond, 10*time.Millisecond)

	// Create an evaluation
	eval, err := s.createEvaluation()
	s.Require().NoError(err)

	// no evaluations should be enqueued
	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "evaluation should not be enqueued")
}

func (s *WatcherTestSuite) createEvaluation() (*models.Evaluation, error) {
	// Create a job
	job := mock.Job()
	eval := mock.Eval()
	eval.JobID = job.ID
	eval.Type = job.Type

	// Create job
	err := s.store.CreateJob(context.Background(), *job, models.Event{})
	s.Require().NoError(err)

	// Create an evaluation
	err = s.store.CreateEvaluation(context.Background(), *eval)
	s.Require().NoError(err)
	return eval, err
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
