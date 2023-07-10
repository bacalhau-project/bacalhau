//go:build unit || !integration

package evaluation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/bacalhau-project/bacalhau/pkg/test/wait"
	"github.com/stretchr/testify/suite"
)

var (
	defaultSched = []string{
		model.JobTypeService,
		model.JobTypeBatch,
	}
	defaultBrokerParams = InMemoryBrokerParams{
		VisibilityTimeout:    5 * time.Second,
		InitialRetryDelay:    5 * time.Millisecond,
		SubsequentRetryDelay: 50 * time.Millisecond,
		MaxReceiveCount:      3,
	}
)

type InMemoryBrokerTestSuite struct {
	suite.Suite
	broker *InMemoryBroker
}

func (s *InMemoryBrokerTestSuite) SetupTest() {
	broker, err := NewInMemoryBroker(defaultBrokerParams)
	s.Require().NoError(err)
	s.broker = broker
}

func (s *InMemoryBrokerTestSuite) TearDownTest() {
	if s.broker != nil {
		s.broker.SetEnabled(false)
	}
}

func TestInMemoryBrokerTestSuite(t *testing.T) {
	suite.Run(t, new(InMemoryBrokerTestSuite))
}

func (s *InMemoryBrokerTestSuite) TestEnqueue_Dequeue_Nack_Ack() {
	// Enqueue, but broker is disabled!
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	// Verify nothing was done
	stats := s.broker.Stats()
	if stats.TotalReady != 0 {
		s.FailNow("bad: %#v", stats)
	}
	s.Require().False(s.broker.Enabled(), "should not be enabled")
	s.Require().Equal(0, stats.TotalReady, "should be zero")
	s.Require().Empty(s.broker.ready, "ready queue should be empty")

	// Enable the broker, and enqueue
	s.broker.SetEnabled(true)
	s.Require().True(s.broker.Enabled(), "should be enabled")
	s.Require().NoError(s.broker.Enqueue(eval))

	// Double enqueue is a no-op
	s.Require().NoError(s.broker.Enqueue(eval))

	// Verify enqueue is done
	stats = s.broker.Stats()
	s.Require().Equal(1, stats.TotalReady, "should have one ready")
	s.Require().Equal(1, stats.ByScheduler[eval.Type].Ready, "should have one ready")

	// Dequeue should work
	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out)

	receiptHandleOut, ok := s.broker.Inflight(out.ID)
	s.Require().True(ok, "should be outstanding")
	s.Require().Equal(receiptHandle, receiptHandleOut)

	// InflightExtend should verify the receiptHandle
	err = s.broker.InflightExtend("nope", "foo")
	s.Require().ErrorIs(err, ErrNotInflight)
	err = s.broker.InflightExtend(out.ID, "foo")
	s.Require().ErrorIs(err, ErrReceiptHandleMismatch)
	err = s.broker.InflightExtend(out.ID, receiptHandleOut)
	s.Require().NoError(err)

	// Check the stats
	stats = s.broker.Stats()
	s.Require().Equal(0, stats.TotalReady)
	s.Require().Equal(1, stats.TotalInflight)
	s.Require().Equal(0, stats.ByScheduler[eval.Type].Ready)
	s.Require().Equal(1, stats.ByScheduler[eval.Type].Inflight)

	// Nack with wrong receiptHandle should fail
	s.Require().Error(s.broker.Nack(eval.ID, "foobarbaz"))

	// Nack back into the queue
	s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle))

	_, ok = s.broker.Inflight(out.ID)
	s.Require().False(ok, "should not be outstanding")

	// Check the stats
	s.Require().NoError(wait.For(wait.Params{Condition: func() (bool, error) {
		stats = s.broker.Stats()
		if stats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalInflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalWaiting != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Ready != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Inflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}

		return true, nil
	}}))

	// Dequeue should work again
	out2, receiptHandle2, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out2)
	s.Require().NotEqual(receiptHandle2, receiptHandle, "should get a new receipt handle")

	receiptHandleOut2, ok := s.broker.Inflight(out.ID)
	s.Require().True(ok, "should be outstanding")
	s.Require().Equal(receiptHandle2, receiptHandleOut2)

	// Ack with wrong receiptHandle
	s.Require().Error(s.broker.Ack(eval.ID, "zip"))

	// Ack finally
	s.Require().NoError(s.broker.Ack(eval.ID, receiptHandle2))
	_, ok = s.broker.Inflight(out.ID)
	s.Require().False(ok, "should not be outstanding")

	// Check the stats
	stats = s.broker.Stats()
	// Check the stats
	stats = s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

func (s *InMemoryBrokerTestSuite) TestNack_Delay() {
	// Enqueue,
	s.broker.SetEnabled(true)
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	// Dequeue should work
	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out)

	// Nack back into the queue
	s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle))

	_, ok := s.broker.Inflight(out.ID)
	s.Require().False(ok, "should not be outstanding")

	// Check the stats to ensure that it is waiting
	stats := s.broker.Stats()
	s.Require().Equal(0, stats.TotalReady)
	s.Require().Equal(0, stats.TotalInflight)
	s.Require().Equal(1, stats.TotalWaiting)
	s.Require().Equal(0, stats.ByScheduler[eval.Type].Ready)
	s.Require().Equal(0, stats.ByScheduler[eval.Type].Inflight)

	// Now wait for it to be re-enqueued
	s.Require().NoError(wait.For(wait.Params{Condition: func() (bool, error) {
		stats = s.broker.Stats()
		if stats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalInflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalWaiting != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Ready != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Inflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}

		return true, nil
	}}))

	// Dequeue should work again
	out2, receiptHandle2, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out2)
	s.Require().NotEqual(receiptHandle2, receiptHandle, "should get a new receipt handle")

	// Capture the time
	start := time.Now()

	// Nack back into the queue
	s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle2))

	// Now wait for it to be re-enqueued
	s.Require().NoError(wait.For(wait.Params{Condition: func() (bool, error) {
		stats = s.broker.Stats()
		if stats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalInflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalWaiting != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Ready != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.ByScheduler[eval.Type].Inflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}

		return true, nil
	}}))

	delay := time.Now().Sub(start)
	s.Require().GreaterOrEqual(delay, s.broker.subsequentNackDelay, "should have waited for the nack delay")

	// Dequeue should work again
	out3, receiptHandle3, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out3)
	s.Require().NotEqual(receiptHandle3, receiptHandle, "should get a new receipt handle")
	s.Require().NotEqual(receiptHandle3, receiptHandle2, "should get a new receipt handle")

	// Ack finally
	s.Require().NoError(s.broker.Ack(eval.ID, receiptHandle3))
	_, ok = s.broker.Inflight(out.ID)
	s.Require().False(ok, "should not be outstanding")

	// Check the stats
	stats = s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

func (s *InMemoryBrokerTestSuite) TestSerialize_DuplicateJobID() {
	s.broker.SetEnabled(true)

	ns1 := "namespace-one"
	ns2 := "namespace-two"
	jobID := "example"

	newEval := func(idx uint64, ns string) *models.Evaluation {
		eval := mock.Eval()
		eval.ID = fmt.Sprintf("eval:%d", idx)
		eval.JobID = jobID
		eval.Namespace = ns
		eval.CreateIndex = idx
		eval.ModifyIndex = idx
		s.Require().NoError(s.broker.Enqueue(eval))
		return eval
	}

	// first job
	eval1 := newEval(1, ns1)
	newEval(2, ns1)
	newEval(3, ns1)
	eval4 := newEval(4, ns1)

	// second job
	eval5 := newEval(5, ns2)
	newEval(6, ns2)
	eval7 := newEval(7, ns2)

	// retreive the stats from the broker, less some stats that aren't
	// interesting for this test and make the test much more verbose
	// to include
	getStats := func() BrokerStats {
		s.T().Helper()
		stats := s.broker.Stats()
		stats.DelayedEvals = nil
		stats.ByScheduler = nil
		return *stats
	}

	s.Require().Equal(
		BrokerStats{TotalReady: 2, TotalInflight: 0, TotalPending: 5, TotalCancelable: 0}, getStats())

	// Dequeue should get 1st eval
	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(out, eval1, "expected 1st eval")

	s.Require().Equal(
		BrokerStats{TotalReady: 1, TotalInflight: 1, TotalPending: 5, TotalCancelable: 0}, getStats())

	// Ack should clear the first eval
	s.Require().NoError(s.broker.Ack(eval1.ID, receiptHandle))
	// eval4 and eval5 are ready
	// eval6 and eval7 are pending
	// eval2 and eval3 are canceled
	s.Require().Equal(
		BrokerStats{TotalReady: 2, TotalInflight: 0, TotalPending: 2, TotalCancelable: 2}, getStats())

	// Dequeue should get 4th eval
	out, receiptHandle, err = s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(out, eval4, "expected 4th eval")
	s.Require().Equal(
		BrokerStats{TotalReady: 1, TotalInflight: 1, TotalPending: 2, TotalCancelable: 2}, getStats())

	// Ack should clear the rest of namespace-one pending but leave
	// namespace-two untouched
	s.Require().NoError(s.broker.Ack(eval4.ID, receiptHandle))
	s.Require().Equal(
		BrokerStats{TotalReady: 1, TotalInflight: 0, TotalPending: 2, TotalCancelable: 2}, getStats())

	// Dequeue should get 5th eval
	out, receiptHandle, err = s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(out, eval5, "expected 5th eval")
	s.Require().Equal(
		BrokerStats{TotalReady: 0, TotalInflight: 1, TotalPending: 2, TotalCancelable: 2}, getStats())

	// Ack should clear remaining namespace-two pending evals
	s.Require().NoError(s.broker.Ack(eval5.ID, receiptHandle))
	s.Require().Equal(
		BrokerStats{TotalReady: 1, TotalInflight: 0, TotalPending: 0, TotalCancelable: 3}, getStats())

	// Dequeue should get 7th eval because that's all that's left
	out, receiptHandle, err = s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(out, eval7, "expected 7th eval")
	s.Require().Equal(
		BrokerStats{TotalReady: 0, TotalInflight: 1, TotalPending: 0, TotalCancelable: 3}, getStats())

	// Last ack should leave the broker empty except for cancels
	s.Require().NoError(s.broker.Ack(eval7.ID, receiptHandle))
	s.Require().Equal(
		BrokerStats{TotalReady: 0, TotalInflight: 0, TotalPending: 0, TotalCancelable: 3}, getStats())
}

func (s *InMemoryBrokerTestSuite) TestEnqueue_Disable() {
	// Enqueue
	eval := mock.Eval()
	s.broker.SetEnabled(true)
	s.Require().NoError(s.broker.Enqueue(eval))

	// Flush via SetEnabled
	s.broker.SetEnabled(false)

	// Check the stats
	stats := s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

func (s *InMemoryBrokerTestSuite) TestEnqueue_Disable_Delay() {
	baseEval := mock.Eval()
	s.broker.SetEnabled(true)

	// Enqueue
	waitEval := baseEval.Copy()
	waitEval.WaitUntil = time.Now().Add(30 * time.Second)
	s.Require().NoError(s.broker.Enqueue(baseEval.Copy()))
	s.Require().NoError(s.broker.Enqueue(waitEval))

	// Flush via SetEnabled
	s.broker.SetEnabled(false)

	// Check the stats
	stats := s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)

	// Enqueue again now we're disabled
	waitEval = baseEval.Copy()
	waitEval.WaitUntil = time.Now().Add(30 * time.Second)
	s.Require().NoError(s.broker.Enqueue(baseEval.Copy()))
	s.Require().NoError(s.broker.Enqueue(waitEval))

	// Check the stats again
	stats = s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

func (s *InMemoryBrokerTestSuite) TestDequeue_Timeout() {
	s.broker.SetEnabled(true)

	start := time.Now()
	dequeueTimeout := 5 * time.Millisecond
	out, _, err := s.broker.Dequeue(defaultSched, dequeueTimeout)
	end := time.Now()

	s.Require().NoError(err)
	s.Require().Nil(out)

	buffer := 20 * time.Millisecond // ci windows is slow
	s.Require().GreaterOrEqual(end.Sub(start), dequeueTimeout, "dequeue too fast")
	s.Require().Less(end.Sub(start), dequeueTimeout+buffer, "dequeue too slow")
}

func (s *InMemoryBrokerTestSuite) TestDequeue_Empty_Timeout() {
	s.broker.SetEnabled(true)

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		out, _, err := s.broker.Dequeue(defaultSched, 0)
		if err != nil {
			errCh <- err
			return
		}
		if out == nil {
			errCh <- errors.New("expected a non-nil value")
			return
		}
	}()

	// Sleep for a little bit
	select {
	case <-time.After(5 * time.Millisecond):
	case err := <-errCh:
		s.Require().NoError(err, "error from dequeue goroutine")
		s.FailNow("Dequeue(0) should block, not finish")
	}

	// Enqueue to unblock the dequeue.
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	select {
	case err := <-errCh:
		s.Require().NoError(err, "error from dequeue goroutine")
	case <-time.After(5 * time.Millisecond):
		s.FailNow("timeout: Dequeue(0) should return after enqueue")
	}
}

// Ensure higher priority dequeued first
func (s *InMemoryBrokerTestSuite) TestDequeue_Priority() {
	s.broker.SetEnabled(true)

	eval1 := mock.Eval()
	eval1.Priority = 10
	s.Require().NoError(s.broker.Enqueue(eval1))

	eval2 := mock.Eval()
	eval2.Priority = 30
	s.Require().NoError(s.broker.Enqueue(eval2))

	eval3 := mock.Eval()
	eval3.Priority = 20
	s.Require().NoError(s.broker.Enqueue(eval3))

	out1, _, _ := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().Equal(out1, eval2, "expected eval2 to be dequeued first")

	out2, _, _ := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().Equal(out2, eval3, "expected eval3 to be dequeued second")

	out3, _, _ := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().Equal(out3, eval1, "expected eval1 to be dequeued last")
}

// Ensure FIFO at fixed priority
func (s *InMemoryBrokerTestSuite) TestDequeue_FIFO() {
	s.broker.SetEnabled(true)
	NUM := 100

	for i := NUM; i > 0; i-- {
		eval := mock.Eval()
		eval.CreateIndex = uint64(i)
		eval.ModifyIndex = uint64(i)
		s.Require().NoError(s.broker.Enqueue(eval))
	}

	for i := 1; i < NUM; i++ {
		out1, _, _ := s.broker.Dequeue(defaultSched, time.Second)
		s.Require().Equal(uint64(i), out1.CreateIndex,
			"eval was not FIFO by CreateIndex",
		)
	}
}

// Ensure fairness between schedulers
func (s *InMemoryBrokerTestSuite) TestDequeue_Fairness() {
	s.broker.SetEnabled(true)
	NUM := 1000

	for i := 0; i < NUM; i++ {
		eval := mock.Eval()
		if i < (NUM / 2) {
			eval.Type = model.JobTypeService
		} else {
			eval.Type = model.JobTypeBatch
		}
		s.Require().NoError(s.broker.Enqueue(eval))
	}

	typeServiceCounter := 0
	typeBatchCounter := 0
	for i := 0; i < NUM; i++ {
		out1, _, _ := s.broker.Dequeue(defaultSched, time.Second)

		switch out1.Type {
		case model.JobTypeService:
			typeServiceCounter += 1
		case model.JobTypeBatch:
			typeBatchCounter += 1
		}
	}
	s.Require().InDeltaf(typeBatchCounter, typeServiceCounter, 250, "expected schedulers to be fair")
}

// Ensure we get unblocked
func (s *InMemoryBrokerTestSuite) TestDequeue_Blocked() {
	s.broker.SetEnabled(true)
	enqueueAfter := 5 * time.Millisecond

	// Start with a blocked dequeue
	outCh := make(chan *models.Evaluation)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		defer close(outCh)
		start := time.Now()
		out, _, err := s.broker.Dequeue(defaultSched, time.Second)
		if err != nil {
			errCh <- err
			return
		}
		end := time.Now()
		buffer := 25 * time.Millisecond // ci windows is slow
		s.Require().GreaterOrEqual(end.Sub(start), enqueueAfter, "dequeue too fast")
		s.Require().Less(end.Sub(start), enqueueAfter+buffer, "dequeue too slow")
		outCh <- out
	}()

	// Wait for a bit, or t.Fatal if an error has already happened in
	// the goroutine
	select {
	case <-time.After(enqueueAfter):
		// no errors yet, soldier on
	case err := <-errCh:
		s.Require().NoError(err, "error from anonymous goroutine before enqueue")
	}

	// Enqueue
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	// Ensure dequeue
	select {
	case out := <-outCh:
		s.Require().Equal(out, eval, "dequeue result mismatch")
	case err := <-errCh:
		s.Require().NoError(err, "error from anonymous goroutine after enqueue")
	case <-time.After(time.Second):
		s.FailNow("timeout waiting for dequeue result")
	}
}

// Ensure we nack in a timely manner
func (s *InMemoryBrokerTestSuite) TestNack_Timeout() {
	s.broker.visibilityTimeout = 5 * time.Millisecond
	s.broker.SetEnabled(true)

	// Enqueue
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	// Dequeue
	out, _, err := s.broker.Dequeue(defaultSched, time.Second)
	start := time.Now()
	s.Require().NoError(err)
	s.Require().Equal(out, eval, "dequeue result mismatch")

	// Dequeue, should block on Nack timer
	out, _, err = s.broker.Dequeue(defaultSched, time.Second)
	end := time.Now()
	s.Require().NoError(err)
	s.Require().Equal(eval, out, "dequeue result mismatch")

	// Check the nack timer
	expectedDelay := s.broker.visibilityTimeout + s.broker.initialNackDelay
	buffer := 30 * time.Millisecond // ci windows is slow
	s.Require().GreaterOrEqual(end.Sub(start), expectedDelay, "dequeue too fast")
	s.Require().Less(end.Sub(start), expectedDelay+buffer, "dequeue too slow")
}

// Ensure we nack in a timely manner
func (s *InMemoryBrokerTestSuite) TestNack_TimeoutReset() {
	s.broker.visibilityTimeout = 50 * time.Millisecond
	s.broker.SetEnabled(true)

	// Enqueue
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	// Dequeue
	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out, "dequeue result mismatch")

	// Reset in 20 milliseconds
	time.Sleep(20 * time.Millisecond)
	start := time.Now()
	s.Require().NoError(s.broker.InflightExtend(out.ID, receiptHandle))

	// Dequeue, should block on Nack timer
	out, _, err = s.broker.Dequeue(defaultSched, time.Second)
	end := time.Now()
	s.Require().NoError(err)
	s.Require().Equal(eval, out, "dequeue result mismatch")

	// Check the nack timer
	buffer := 30 * time.Millisecond
	s.Require().GreaterOrEqual(end.Sub(start), s.broker.visibilityTimeout, "dequeue too fast after nack timeout reset")
	s.Require().Less(end.Sub(start), s.broker.visibilityTimeout+buffer, "dequeue too slow after nack timeout reset")
}

func (s *InMemoryBrokerTestSuite) TestDeliveryLimit() {
	s.broker.SetEnabled(true)

	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	for i := 0; i < defaultBrokerParams.MaxReceiveCount; i++ {
		// Dequeue should work
		out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
		s.Require().NoError(err)
		s.Require().Equal(eval, out)

		s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle))
	}

	// Check the stats
	stats := s.broker.Stats()
	s.Equal(1, stats.TotalReady)
	s.Equal(0, stats.TotalInflight)
	s.Equal(1, stats.ByScheduler[deadLetterQueue].Ready)
	s.Equal(0, stats.ByScheduler[deadLetterQueue].Inflight)

	// Dequeue from failed queue
	out, receiptHandle, err := s.broker.Dequeue([]string{deadLetterQueue}, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out)

	// Check the stats
	stats = s.broker.Stats()
	s.Require().Equal(0, stats.TotalReady)
	s.Require().Equal(1, stats.TotalInflight)
	s.Require().Equal(0, stats.ByScheduler[deadLetterQueue].Ready)
	s.Require().Equal(1, stats.ByScheduler[deadLetterQueue].Inflight)

	// Ack finally
	s.Require().NoError(s.broker.Ack(out.ID, receiptHandle))
	if _, ok := s.broker.Inflight(out.ID); ok {
		s.FailNow("outstanding evals should be empty")
	}

	// Check the stats
	stats = s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

func (s *InMemoryBrokerTestSuite) TestAckAtDeliveryLimit() {
	s.broker.SetEnabled(true)

	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	for i := 0; i < defaultBrokerParams.MaxReceiveCount; i++ {
		// Dequeue should work
		out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
		s.Require().NoError(err)
		s.Require().Equal(eval, out)

		if i == 2 {
			s.Require().NoError(s.broker.Ack(eval.ID, receiptHandle))
		} else {
			s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle))
		}
	}

	// Check the stats
	stats := s.broker.Stats()
	s.Require().True(stats.IsEmpty(), "should be empty stats: %#v", stats)
}

// Ensure that delayed evaluations work as expected
func (s *InMemoryBrokerTestSuite) TestWaitUntil() {
	s.broker.SetEnabled(true)

	now := time.Now()
	// Create a few of evals with WaitUntil set
	eval1 := mock.Eval()
	eval1.WaitUntil = now.Add(1 * time.Second)
	eval1.CreateIndex = 1
	s.Require().NoError(s.broker.Enqueue(eval1))

	eval2 := mock.Eval()
	eval2.WaitUntil = now.Add(100 * time.Millisecond)
	// set CreateIndex to use as a tie breaker when eval2
	// and eval3 are both in the pending evals heap
	eval2.CreateIndex = 2
	s.Require().NoError(s.broker.Enqueue(eval2))

	eval3 := mock.Eval()
	eval3.WaitUntil = now.Add(20 * time.Millisecond)
	eval3.CreateIndex = 1
	s.Require().NoError(s.broker.Enqueue(eval3))
	s.Require().Equal(3, s.broker.stats.TotalWaiting)
	// sleep enough for two evals to be ready
	time.Sleep(200 * time.Millisecond)

	// first dequeue should return eval3
	out, _, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval3, out)

	// second dequeue should return eval2
	out, _, err = s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval2, out)

	// third dequeue should return eval1
	out, _, err = s.broker.Dequeue(defaultSched, 2*time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval1, out)
	s.Require().Equal(0, s.broker.stats.TotalWaiting)
}

// Ensure that priority is taken into account when enqueueing many evaluations.
func (s *InMemoryBrokerTestSuite) TestEnqueueAll_Dequeue_Fair() {
	s.broker.SetEnabled(true)

	// Start with a blocked dequeue
	outCh := make(chan *models.Evaluation)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		defer close(outCh)
		start := time.Now()
		out, _, err := s.broker.Dequeue(defaultSched, time.Second)
		if err != nil {
			errCh <- err
			return
		}
		end := time.Now()
		if d := end.Sub(start); d < 5*time.Millisecond {
			errCh <- fmt.Errorf("test broker dequeue duration too fast: %v", d)
			return
		}
		outCh <- out
	}()

	// Wait for a bit, or t.Fatal if an error has already happened in
	// the goroutine
	select {
	case <-time.After(5 * time.Millisecond):
		// no errors yet, soldier on
	case err := <-errCh:
		s.Require().NoError(err, "error from anonymous goroutine before enqueue")
	}

	// Enqueue
	evals := make(map[*models.Evaluation]string, 8)
	expectedPriority := 90
	for i := 10; i <= expectedPriority; i += 10 {
		eval := mock.Eval()
		eval.Priority = i
		evals[eval] = ""

	}
	s.Require().NoError(s.broker.EnqueueAll(evals))

	// Ensure dequeue
	select {
	case out := <-outCh:
		s.Require().Equal(expectedPriority, out.Priority)
	case err := <-errCh:
		s.Require().NoError(err, "error from anonymous goroutine after enqueue")
	case <-time.After(time.Second):
		s.FailNow("timeout waiting for dequeue result")
	}
}

func (s *InMemoryBrokerTestSuite) TestEnqueueAll_Requeue_Ack() {
	s.broker.SetEnabled(true)

	// Create the evaluation, enqueue and dequeue
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out)

	// Requeue the same evaluation.
	s.Require().NoError(s.broker.EnqueueAll(map[*models.Evaluation]string{eval: receiptHandle}))

	// The stats should show one unacked
	stats := s.broker.Stats()
	s.Require().Equal(0, stats.TotalReady)
	s.Require().Equal(1, stats.TotalInflight)

	// Ack the evaluation.
	s.Require().NoError(s.broker.Ack(eval.ID, receiptHandle))

	// Check stats again as this should cause the re-enqueued one to transition
	// into the ready state
	stats = s.broker.Stats()
	s.Require().Equal(1, stats.TotalReady)
	s.Require().Equal(0, stats.TotalInflight)

	// Another dequeue should be successful
	out2, receiptHandle2, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out2)
	s.Require().NotEqual(receiptHandle, receiptHandle2)
}

func (s *InMemoryBrokerTestSuite) TestEnqueueAll_Requeue_Nack() {
	s.broker.SetEnabled(true)

	// Create the evaluation, enqueue and dequeue
	eval := mock.Eval()
	s.Require().NoError(s.broker.Enqueue(eval))

	out, receiptHandle, err := s.broker.Dequeue(defaultSched, time.Second)
	s.Require().NoError(err)
	s.Require().Equal(eval, out)

	// Requeue the same evaluation.
	s.Require().NoError(s.broker.EnqueueAll(map[*models.Evaluation]string{eval: receiptHandle}))

	// The stats should show one unacked
	stats := s.broker.Stats()
	s.Require().Equal(0, stats.TotalReady)
	s.Require().Equal(1, stats.TotalInflight)

	// Nack the evaluation.
	s.Require().NoError(s.broker.Nack(eval.ID, receiptHandle))

	// Check stats again as this should cause the re-enqueued one to be dropped
	s.Require().NoError(wait.For(wait.Params{Condition: func() (bool, error) {
		stats = s.broker.Stats()
		if stats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if stats.TotalInflight != 0 {
			return false, fmt.Errorf("bad: %#v", stats)
		}
		if len(s.broker.requeue) != 0 {
			return false, fmt.Errorf("bad: %#v", s.broker.requeue)
		}

		return true, nil
	}}))
}

func (s *InMemoryBrokerTestSuite) TestNamespacedJobs() {
	s.broker.SetEnabled(true)

	// Create evals with the same jobid and different namespace
	jobId := "test-jobID"

	eval1 := mock.Eval()
	eval1.JobID = jobId
	eval1.Namespace = "n1"
	s.Require().NoError(s.broker.Enqueue(eval1))

	// This eval should not block
	eval2 := mock.Eval()
	eval2.JobID = jobId
	eval2.Namespace = "default"
	s.Require().NoError(s.broker.Enqueue(eval2))

	// This eval should block
	eval3 := mock.Eval()
	eval3.JobID = jobId
	eval3.Namespace = "default"
	s.Require().NoError(s.broker.Enqueue(eval3))

	out1, _, err := s.broker.Dequeue(defaultSched, 5*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Equal(eval1.ID, out1.ID)

	out2, _, err := s.broker.Dequeue(defaultSched, 5*time.Millisecond)
	s.Require().NoError(err)
	s.Require().Equal(eval2.ID, out2.ID)

	out3, _, err := s.broker.Dequeue(defaultSched, 5*time.Millisecond)
	s.Require().Nil(err)
	s.Require().Nil(out3)

	s.Require().Len(s.broker.pending, 1)

}

func (s *InMemoryBrokerTestSuite) TestCancelable() {
	s.broker.visibilityTimeout = time.Minute

	var evals []*models.Evaluation
	for i := 0; i < 20; i++ {
		eval := mock.Eval()
		evals = append(evals, eval)
	}
	s.broker.cancelable = evals
	s.broker.stats.TotalCancelable = len(s.broker.cancelable)

	s.Require().Len(s.broker.cancelable, 20)
	cancelable := s.broker.Cancelable(10)
	s.Require().Len(cancelable, 10)
	s.Require().Len(s.broker.cancelable, 10)
	s.Require().Equal(10, s.broker.stats.TotalCancelable)

	cancelable = s.broker.Cancelable(20)
	s.Require().Len(cancelable, 10)
	s.Require().Empty(s.broker.cancelable)
	s.Require().Equal(0, s.broker.stats.TotalCancelable)
}
