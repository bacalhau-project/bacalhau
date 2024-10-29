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
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type WatchHandlerTestSuite struct {
	suite.Suite
	store        jobstore.Store
	broker       *evaluation.InMemoryBroker
	registry     watcher.Registry
	watchHandler *evaluation.WatchHandler
	ctx          context.Context
}

func (s *WatchHandlerTestSuite) SetupTest() {
	var err error
	s.ctx = context.Background()
	s.store, err = boltjobstore.NewBoltJobStore(filepath.Join(s.T().TempDir(), "evaluation-watcher.db"))
	s.Require().NoError(err)

	s.broker, err = evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{})
	s.Require().NoError(err)
	s.broker.SetEnabled(true)

	s.registry = watcher.NewRegistry(s.store.GetEventStore())
	s.watchHandler = evaluation.NewWatchHandler(s.broker)

	// Start watching for events
	_, err = s.registry.Watch(s.ctx, "test-watcher", s.watchHandler,
		watcher.WithInitialEventIterator(watcher.LatestIterator()),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectEvaluation},
			Operations:  []watcher.Operation{watcher.OperationCreate},
		}),
	)
	s.Require().NoError(err)
}

func (s *WatchHandlerTestSuite) TearDownTest() {
	_ = s.registry.Stop(s.ctx)
	_ = s.store.Close(s.ctx)
}

func (s *WatchHandlerTestSuite) TestEvaluationEnqueued() {
	// Create an evaluation
	eval, err := s.createEvaluation()
	s.Require().NoError(err)

	// Verify it was enqueued in the broker
	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 100*time.Millisecond)
	s.Require().NoError(err)
	s.Require().NotNil(dequeuedEval, "evaluation should be enqueued")
	s.Equal(eval.ID, dequeuedEval.ID)
	s.Equal(eval.JobID, dequeuedEval.JobID)

	// Verify no more evaluations are queued
	dequeuedEval, _, err = s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "no more evaluations should be enqueued")
}

func (s *WatchHandlerTestSuite) TestEvaluationDeletedIgnored() {
	// Create and then delete an evaluation
	eval, err := s.createEvaluation()
	s.Require().NoError(err)

	// Dequeue the created evaluation
	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 100*time.Millisecond)
	s.Require().NoError(err)
	s.Require().NotNil(dequeuedEval, "evaluation should be enqueued")

	// Delete the evaluation
	err = s.store.DeleteEvaluation(s.ctx, eval.ID)
	s.Require().NoError(err)

	// Verify no new evaluation was enqueued from the delete event
	dequeuedEval, _, err = s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "delete event should not enqueue evaluation")
}

func (s *WatchHandlerTestSuite) TestStoppedWatcher() {
	s.Require().NoError(s.registry.Stop(s.ctx))

	// Create an evaluation after stopping
	eval, err := s.createEvaluation()
	s.Require().NoError(err)

	// Verify no evaluations were enqueued
	dequeuedEval, _, err := s.broker.Dequeue([]string{eval.Type}, 10*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Nil(dequeuedEval, "evaluation should not be enqueued after stopping")
}

func (s *WatchHandlerTestSuite) createEvaluation() (*models.Evaluation, error) {
	job := mock.Job()
	eval := mock.Eval()
	eval.JobID = job.ID
	eval.Type = job.Type

	tx, err := s.store.BeginTx(context.Background())
	s.Require().NoError(err)
	defer func(tx jobstore.TxContext) {
		_ = tx.Rollback()
	}(tx)

	s.Require().NoError(s.store.CreateJob(tx, *job))
	s.Require().NoError(s.store.CreateEvaluation(tx, *eval))
	s.Require().NoError(tx.Commit())
	return eval, nil
}

func TestWatchHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(WatchHandlerTestSuite))
}
