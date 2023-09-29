package evaluation

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/metric"
)

const (
	// deadLetterQueue is the queue we add Evaluations to once
	// they've reached the MaxReceiveCount. This allows a consumer to
	// set the status to failed.
	deadLetterQueue = "_dlq"

	// initialCapacity is the initial capacity of the broker heap per scheduler
	initialCapacity = 16
)

var (
	// ErrNotInflight is returned if an evaluation is not inflight
	ErrNotInflight = errors.New("evaluation is not inflight")

	// ErrReceiptHandleMismatch is the outstanding eval has a different receiptHandle
	ErrReceiptHandleMismatch = errors.New("evaluation receiptHandle does not match")

	// ErrNackTimeoutReached is returned if an expired evaluation is reset
	ErrNackTimeoutReached = errors.New("evaluation visibility timeout reached")
)

// compile-time check to ensure type implements the models.EvaluationBroker interface
var _ orchestrator.EvaluationBroker = &InMemoryBroker{}

type InMemoryBrokerParams struct {
	VisibilityTimeout    time.Duration
	InitialRetryDelay    time.Duration
	SubsequentRetryDelay time.Duration
	MaxReceiveCount      int
}

// InMemoryBroker The broker is designed to be entirely in-memory.
type InMemoryBroker struct {
	visibilityTimeout time.Duration
	maxReceiveCount   int
	enabled           bool

	// evals tracks queued evaluations by ID to de-duplicate enqueue.
	// The counter is the number of times we've attempted delivery,
	// and is used to eventually fail an evaluation.
	evals map[string]int

	// jobEvals tracks queued evaluations by a job's ID and namespace to serialize them
	jobEvals map[models.NamespacedID]string

	// pending tracks the pending evaluations by JobID in a priority queue
	pending map[models.NamespacedID]PendingEvaluations

	// cancelable tracks previously pending evaluations (for any job) that are
	// now safe for the Eval.Ack RPC to cancel in batches
	cancelable []*models.Evaluation

	// ready tracks the ready jobs by scheduler in a priority queue
	ready map[string]ReadyEvaluations

	// inflight is a map of evalID to an un-acknowledged evaluations
	inflight map[string]*inflightEval

	// waiting is used to notify on a per-scheduler basis of ready work
	waiting map[string]chan struct{}

	// requeue tracks evaluations that need to be re-enqueued once the current
	// evaluation finishes by receiptHandle. If the receiptHandle is Nacked or rejected the
	// evaluation is dropped but if Acked successfully, the evaluation is
	// queued.
	requeue map[string]*models.Evaluation

	// delayedEvalCancelFunc is used to stop the long running go routine
	// that processes delayed evaluations
	delayedEvalCancelFunc context.CancelFunc

	// delayHeap is a heap used to track incoming evaluations that are
	// not eligible to enqueue until their WaitTime
	delayHeap *collections.ScheduledTaskHeap[*models.Evaluation]

	// delayedEvalsUpdateCh is used to trigger notifications for updates
	// to the delayHeap
	delayedEvalsUpdateCh chan struct{}

	// initialNackDelay is the delay applied before re-enqueuing a
	// Nacked evaluation for the first time.
	initialNackDelay time.Duration

	// subsequentNackDelay is the delay applied before reenqueuing
	// an evaluation that has been Nacked more than once. This delay is
	// compounding after the first Nack.
	subsequentNackDelay time.Duration

	metricRegistration metric.Registration

	stats *BrokerStats

	l sync.RWMutex
}

// NewInMemoryBroker creates a new evaluation broker. This is parameterized with:
//   - VisibilityTimeout used for messages. If not acknowledged before this time we
//     assume a Nack and attempt to redeliver.
//   - MaxReceiveCount which prevents a failing eval from being endlessly delivered.
//   - InitialNackDelay which is the delay before making a first-time Nacked
//     evaluation available again
//   - SubsequentNackDelay is the compounding delay before making evaluations
//     available again, after the first Nack.
func NewInMemoryBroker(params InMemoryBrokerParams) (*InMemoryBroker, error) {
	if params.VisibilityTimeout < 0 {
		return nil, fmt.Errorf("timeout cannot be negative")
	}
	b := &InMemoryBroker{
		visibilityTimeout:    params.VisibilityTimeout,
		maxReceiveCount:      params.MaxReceiveCount,
		enabled:              false,
		stats:                new(BrokerStats),
		evals:                make(map[string]int),
		jobEvals:             make(map[models.NamespacedID]string),
		pending:              make(map[models.NamespacedID]PendingEvaluations),
		cancelable:           []*models.Evaluation{},
		ready:                make(map[string]ReadyEvaluations),
		inflight:             make(map[string]*inflightEval),
		waiting:              make(map[string]chan struct{}),
		requeue:              make(map[string]*models.Evaluation),
		initialNackDelay:     params.InitialRetryDelay,
		subsequentNackDelay:  params.SubsequentRetryDelay,
		delayHeap:            collections.NewScheduledTaskHeap[*models.Evaluation](),
		delayedEvalsUpdateCh: make(chan struct{}, 1),
	}
	b.stats.ByScheduler = make(map[string]*SchedulerStats)
	b.stats.DelayedEvals = make(map[string]*models.Evaluation)

	return b, nil
}

// Enabled is used to check if the broker is enabled.
func (b *InMemoryBroker) Enabled() bool {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.enabled
}

// SetEnabled is used to control if the broker is enabled.
func (b *InMemoryBroker) SetEnabled(enabled bool) {
	b.l.Lock()
	defer b.l.Unlock()

	prevEnabled := b.enabled
	b.enabled = enabled
	if !prevEnabled && enabled {
		// start the go routine for delayed evals
		ctx, cancel := context.WithCancel(context.Background())
		b.delayedEvalCancelFunc = cancel
		go b.runDelayedEvalsWatcher(ctx, b.delayedEvalsUpdateCh)

		metricRegistration, err := b.registerMetrics()
		if err != nil {
			log.Error().Err(err).Msg("failed to register metrics. Evaluation metrics will not be available")
		} else {
			b.metricRegistration = metricRegistration
		}
	}

	if !enabled {
		b.flush()
	}
}

func (b *InMemoryBroker) Enqueue(evaluation *models.Evaluation) error {
	b.l.Lock()
	defer b.l.Unlock()
	return b.processEnqueue(evaluation, "")
}

func (b *InMemoryBroker) EnqueueAll(evals map[*models.Evaluation]string) error {
	// The lock needs to be held until all evaluations are enqueued. This is so
	// that when Dequeue operations are unblocked they will pick the highest
	// priority evaluations.
	b.l.Lock()
	defer b.l.Unlock()
	for eval, receiptHandle := range evals {
		err := b.processEnqueue(eval, receiptHandle)
		if err != nil {
			return err
		}
	}
	return nil
}

// processEnqueue deduplicates evals and either enqueue immediately or enforce
// the evals wait time. If the receiptHandle is passed, and the evaluation ID is
// outstanding, the evaluation is blocked until an Ack/Nack is received.
// processEnqueue must be called with the lock held.
func (b *InMemoryBroker) processEnqueue(eval *models.Evaluation, receiptHandle string) error {
	// If we're not enabled, don't enable more queuing.
	if !b.enabled {
		log.Debug().Msgf("broker is not enabled, dropping evaluation %s for job %s", eval.ID, eval.JobID)
		return nil
	}
	log.Debug().Msgf("enqueueing evaluation %s for job %s, triggered by: %s", eval.ID, eval.JobID, eval.TriggeredBy)

	// Check if already enqueued
	if _, ok := b.evals[eval.ID]; ok {
		if receiptHandle == "" {
			return nil
		}

		// If the receiptHandle has been passed, the evaluation is being reblocked by
		// the scheduler and should be processed once the outstanding evaluation
		// is Acked or Nacked.
		if inflight, ok := b.inflight[eval.ID]; ok && inflight.ReceiptHandle == receiptHandle {
			b.requeue[receiptHandle] = eval
		}
		return nil
	} else {
		b.evals[eval.ID] = 0
	}

	return b.enqueueLocked(eval, eval.Type)
}

// enqueueLocked is used to enqueue with the lock held
func (b *InMemoryBroker) enqueueLocked(eval *models.Evaluation, queueName string) (err error) {
	// Do nothing if not enabled
	if !b.enabled {
		return
	}
	if eval.WaitUntil.After(time.Now().UTC()) {
		err = b.enqueueWaiting(eval)
	} else {
		b.enqueueReady(eval, queueName)
	}
	return err
}

func (b *InMemoryBroker) enqueueWaiting(eval *models.Evaluation) error {
	err := b.delayHeap.Push(&evalWrapper{eval})
	if err != nil {
		return err
	}
	b.stats.TotalWaiting += 1
	b.stats.DelayedEvals[eval.ID] = eval
	// Signal an update.
	select {
	case b.delayedEvalsUpdateCh <- struct{}{}:
	default:
	}
	return nil
}

// enqueueReady is used to enqueue with the lock held
func (b *InMemoryBroker) enqueueReady(eval *models.Evaluation, queueName string) {
	// Check if there is a ready evaluation for this JobID
	namespacedID := models.NamespacedID{
		ID:        eval.JobID,
		Namespace: eval.Namespace,
	}
	readyEval := b.jobEvals[namespacedID]
	if readyEval == "" {
		b.jobEvals[namespacedID] = eval.ID
	} else if readyEval != eval.ID {
		pending := b.pending[namespacedID]
		heap.Push(&pending, eval)
		b.pending[namespacedID] = pending
		b.stats.TotalPending += 1
		return
	}

	// Find the next ready eval by scheduler class
	readyQueue, ok := b.ready[queueName]
	if !ok {
		readyQueue = make([]*models.Evaluation, 0, initialCapacity)
		if _, exist := b.waiting[queueName]; !exist {
			b.waiting[queueName] = make(chan struct{}, 1)
		}
	}

	// Push onto the heap
	heap.Push(&readyQueue, eval)
	b.ready[queueName] = readyQueue

	// Update the stats
	b.stats.TotalReady += 1
	bySched, ok := b.stats.ByScheduler[queueName]
	if !ok {
		bySched = &SchedulerStats{}
		b.stats.ByScheduler[queueName] = bySched
	}
	bySched.Ready += 1

	// Unblock any pending dequeues
	select {
	case b.waiting[queueName] <- struct{}{}:
	default:
	}
}

func (b *InMemoryBroker) Dequeue(types []string, timeout time.Duration) (*models.Evaluation, string, error) {
	var timeoutTimer *time.Timer
	var timeoutCh <-chan time.Time
SCAN:
	// Scan for work
	eval, receiptHandle, err := b.scanForSchedulers(types)
	if err != nil {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}
		return nil, "", err
	}

	// Check if we have something
	if eval != nil {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}
		return eval, receiptHandle, nil
	}

	// Setup the timeout channel the first time around
	if timeoutTimer == nil && timeout != 0 {
		timeoutTimer = time.NewTimer(timeout)
		timeoutCh = timeoutTimer.C
	}

	// Block until we get work
	scan := b.waitForSchedulers(types, timeoutCh)
	if scan {
		goto SCAN
	}
	return nil, "", nil
}

// scanForSchedulers scans for work on any of the schedulers. The highest priority work
// is dequeued first. This may return nothing if there is no work waiting.
func (b *InMemoryBroker) scanForSchedulers(types []string) (*models.Evaluation, string, error) {
	b.l.Lock()
	defer b.l.Unlock()

	// Do nothing if not enabled
	if !b.enabled {
		return nil, "", fmt.Errorf("eval broker disabled")
	}

	// Scan for eligible work
	var eligibleSched []string
	var eligiblePriority int
	for _, sched := range types {
		// Get the ready queue for this scheduler
		readyQueue, ok := b.ready[sched]
		if !ok {
			continue
		}

		// Peek at the next item
		ready := readyQueue.Peek()
		if ready == nil {
			continue
		}

		// Add to eligible if equal or greater priority
		if len(eligibleSched) == 0 || ready.Priority > eligiblePriority {
			eligibleSched = []string{sched}
			eligiblePriority = ready.Priority
		} else if eligiblePriority > ready.Priority {
			continue
		} else if eligiblePriority == ready.Priority {
			eligibleSched = append(eligibleSched, sched)
		}
	}

	// Determine behavior based on eligible work
	switch n := len(eligibleSched); n {
	case 0:
		// No work to do!
		return nil, "", nil

	case 1:
		// Only a single task, dequeue
		return b.dequeueForSched(eligibleSched[0])

	default:
		// Multiple tasks. We pick a random task so that we fairly
		// distribute work.
		offset := rand.Intn(n) // #nosec
		return b.dequeueForSched(eligibleSched[offset])
	}
}

// dequeueForSched is used to dequeue the next work item for a given scheduler.
// This assumes locks are held and that this scheduler has work
func (b *InMemoryBroker) dequeueForSched(jobType string) (*models.Evaluation, string, error) {
	readyQueue := b.ready[jobType]
	raw := heap.Pop(&readyQueue)
	b.ready[jobType] = readyQueue
	eval := raw.(*models.Evaluation)

	// Generate a UUID for the receipt handle
	receiptHandle := uuid.NewString()

	// Setup Nack timer
	nackTimer := time.AfterFunc(b.visibilityTimeout, func() {
		err := b.Nack(eval.ID, receiptHandle)
		if err != nil {
			log.Error().Err(err).Msgf("failed to nack expired evaluation %s", eval.ID)
		}
	})

	// Add to the inflight queue
	b.inflight[eval.ID] = &inflightEval{
		Eval:            eval,
		ReceiptHandle:   receiptHandle,
		VisibilityTimer: nackTimer,
	}

	// Increment the dequeue count
	b.evals[eval.ID] += 1

	// Update the stats
	b.stats.TotalReady -= 1
	b.stats.TotalInflight += 1
	bySched := b.stats.ByScheduler[jobType]
	bySched.Ready -= 1
	bySched.Inflight += 1

	return eval, receiptHandle, nil
}

// waitForSchedulers is used to wait for work on any of the scheduler or until a timeout.
// Returns if there is work waiting potentially.
func (b *InMemoryBroker) waitForSchedulers(types []string, timeoutCh <-chan time.Time) bool {
	doneCh := make(chan struct{})
	readyCh := make(chan struct{}, 1)
	defer close(doneCh)

	// Start all the watchers
	b.l.Lock()
	for _, sched := range types {
		waitCh, ok := b.waiting[sched]
		if !ok {
			waitCh = make(chan struct{}, 1)
			b.waiting[sched] = waitCh
		}

		// Start a goroutine that either waits for the waitCh on this scheduler
		// to unblock or for this waitForSchedulers call to return
		go func() {
			select {
			// TODO: what happens if two routines pull from their wait ch at the exact same time?
			case <-waitCh:
				select {
				case readyCh <- struct{}{}:
				default:
				}
			case <-doneCh:
			}
		}()
	}
	b.l.Unlock()

	// Block until we have ready work and should scan, or until we timeout
	// and should not make an attempt to scan for work
	select {
	case <-readyCh:
		return true
	case <-timeoutCh:
		return false
	}
}

func (b *InMemoryBroker) Inflight(evalID string) (string, bool) {
	b.l.RLock()
	defer b.l.RUnlock()
	inflight, ok := b.inflight[evalID]
	if !ok {
		return "", false
	}
	return inflight.ReceiptHandle, true
}

func (b *InMemoryBroker) InflightExtend(evalID, receiptHandle string) error {
	b.l.RLock()
	defer b.l.RUnlock()
	inflight, ok := b.inflight[evalID]
	if !ok {
		return ErrNotInflight
	}
	if inflight.ReceiptHandle != receiptHandle {
		return ErrReceiptHandleMismatch
	}
	if !inflight.VisibilityTimer.Reset(b.visibilityTimeout) {
		return ErrNackTimeoutReached
	}
	return nil
}

func (b *InMemoryBroker) Ack(evalID, receiptHandle string) error {
	b.l.Lock()
	defer b.l.Unlock()

	// Always delete the requeued evaluation. Either the Ack is successful and
	// we requeue it or it isn't and we want to remove it.
	defer delete(b.requeue, receiptHandle)

	// Lookup the inflight eval
	inflight, ok := b.inflight[evalID]
	if !ok {
		return fmt.Errorf("evaluation ID not found")
	}
	if inflight.ReceiptHandle != receiptHandle {
		return fmt.Errorf("receiptHandle does not match for Evaluation ID")
	}

	// Ensure we were able to stop the timer
	if !inflight.VisibilityTimer.Stop() {
		return fmt.Errorf("evaluation ID Ack'd after Nack timer expiration")
	}

	// Update the stats
	b.stats.TotalInflight -= 1
	queue := inflight.Eval.Type
	if b.evals[evalID] > b.maxReceiveCount {
		queue = deadLetterQueue
	}
	bySched := b.stats.ByScheduler[queue]
	bySched.Inflight -= 1

	// Cleanup
	delete(b.inflight, evalID)
	delete(b.evals, evalID)

	namespacedID := models.NamespacedID{
		ID:        inflight.Eval.JobID,
		Namespace: inflight.Eval.Namespace,
	}
	delete(b.jobEvals, namespacedID)

	// Check if there are any pending evaluations
	if pending := b.pending[namespacedID]; len(pending) != 0 {
		// Only enqueue the latest pending evaluation and cancel the rest
		cancelable := pending.MarkForCancel()
		b.cancelable = append(b.cancelable, cancelable...)
		b.stats.TotalCancelable = len(b.cancelable)
		b.stats.TotalPending -= len(cancelable)

		// If any remain, enqueue an eval
		if len(pending) > 0 {
			eval := heap.Pop(&pending).(*models.Evaluation)
			b.stats.TotalPending -= 1
			err := b.enqueueLocked(eval, eval.Type)
			if err != nil {
				return err
			}
		}

		// Clean up if there are no more after that
		if len(pending) > 0 {
			b.pending[namespacedID] = pending
		} else {
			delete(b.pending, namespacedID)
		}
	}

	// Re-enqueue the evaluation.
	if eval, ok := b.requeue[receiptHandle]; ok {
		err := b.processEnqueue(eval, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *InMemoryBroker) Nack(evalID, receiptHandle string) error {
	b.l.Lock()
	defer b.l.Unlock()

	// Always delete the requeued evaluation since the Nack means the requeue is
	// invalid.
	delete(b.requeue, receiptHandle)

	// Lookup the inflight eval
	inflight, ok := b.inflight[evalID]
	if !ok {
		return fmt.Errorf("evaluation ID not found")
	}
	if inflight.ReceiptHandle != receiptHandle {
		return fmt.Errorf("receiptHandle does not match for Evaluation ID")
	}

	// Stop the timer, doesn't matter if we've missed it
	inflight.VisibilityTimer.Stop()

	// Cleanup
	delete(b.inflight, evalID)

	// Update the stats
	b.stats.TotalInflight -= 1
	bySched := b.stats.ByScheduler[inflight.Eval.Type]
	bySched.Inflight -= 1

	// Check if we've hit the delivery limit, and re-enqueue
	// in the failedQueue
	dequeues := b.evals[evalID]
	e := inflight.Eval
	var queue string
	if dequeues >= b.maxReceiveCount {
		log.Debug().Msgf("Nack: %s has been dequeued %d times, moving to failedQueue", evalID, dequeues)
		queue = deadLetterQueue
	} else {
		queue = e.Type
		e.WaitUntil = time.Now().Add(b.nackReenqueueDelay(e, dequeues)).UTC()
	}
	return b.enqueueLocked(e, queue)
}

// nackReenqueueDelay is used to determine the delay that should be applied on
// the evaluation given the number of previous attempts
func (b *InMemoryBroker) nackReenqueueDelay(eval *models.Evaluation, prevDequeues int) time.Duration {
	switch {
	case prevDequeues <= 0:
		return 0
	case prevDequeues == 1:
		return b.initialNackDelay
	default:
		// For each subsequent nack compound a delay
		return time.Duration(prevDequeues-1) * b.subsequentNackDelay
	}
}

// Flush is used to clear the state of the broker. It must be called from within
// the lock.
func (b *InMemoryBroker) flush() {
	// Unblock any waiters
	for _, waitCh := range b.waiting {
		close(waitCh)
	}

	// Cancel any Nack timers
	for _, inflight := range b.inflight {
		inflight.VisibilityTimer.Stop()
	}

	// Cancel the delayed evaluations goroutine
	if b.delayedEvalCancelFunc != nil {
		b.delayedEvalCancelFunc()
	}

	if b.metricRegistration != nil {
		_ = b.metricRegistration.Unregister()
		b.metricRegistration = nil
	}

	// Clear out the update channel for delayed evaluations
	b.delayedEvalsUpdateCh = make(chan struct{}, 1)

	// Reset the broker
	b.stats.TotalReady = 0
	b.stats.TotalInflight = 0
	b.stats.TotalPending = 0
	b.stats.TotalWaiting = 0
	b.stats.TotalCancelable = 0
	b.stats.DelayedEvals = make(map[string]*models.Evaluation)
	b.stats.ByScheduler = make(map[string]*SchedulerStats)
	b.evals = make(map[string]int)
	b.jobEvals = make(map[models.NamespacedID]string)
	b.pending = make(map[models.NamespacedID]PendingEvaluations)
	b.cancelable = []*models.Evaluation{}
	b.ready = make(map[string]ReadyEvaluations)
	b.inflight = make(map[string]*inflightEval)
	b.waiting = make(map[string]chan struct{})
	b.delayHeap = collections.NewScheduledTaskHeap[*models.Evaluation]()
}

// runDelayedEvalsWatcher is a long-lived function that waits till a time
// deadline is met for pending evaluations before enqueuing them
func (b *InMemoryBroker) runDelayedEvalsWatcher(ctx context.Context, updateCh <-chan struct{}) {
	var timerChannel <-chan time.Time
	var delayTimer *time.Timer
	for {
		eval, waitUntil := b.nextDelayedEval()
		if waitUntil.IsZero() {
			timerChannel = nil
		} else {
			launchDur := waitUntil.Sub(time.Now().UTC())
			if delayTimer == nil {
				delayTimer = time.NewTimer(launchDur)
			} else {
				delayTimer.Reset(launchDur)
			}
			timerChannel = delayTimer.C
		}

		select {
		case <-ctx.Done():
			return
		case <-timerChannel:
			// remove from the heap since we can enqueue it now
			b.l.Lock()
			log.Debug().Msgf("Enqueuing delayed eval %s", eval.ID)
			b.delayHeap.Remove(&evalWrapper{eval})
			b.stats.TotalWaiting -= 1
			delete(b.stats.DelayedEvals, eval.ID)
			b.enqueueReady(eval, eval.Type)
			b.l.Unlock()
		case <-updateCh:
			continue
		}
	}
}

// nextDelayedEval returns the next delayed eval to launch and when it should be enqueued.
// This peeks at the heap to return the top, where the top is the item with the shortest wait.
// If the heap is empty, this returns nil and zero time.
func (b *InMemoryBroker) nextDelayedEval() (*models.Evaluation, time.Time) {
	b.l.RLock()
	defer b.l.RUnlock()

	// If there is nothing wait for an update.
	if b.delayHeap.Length() == 0 {
		return nil, time.Time{}
	}
	nextEval := b.delayHeap.Peek()
	if nextEval == nil {
		return nil, time.Time{}
	}
	eval := nextEval.Data()
	return eval, nextEval.WaitUntil()
}

// Stats is used to query the state of the broker
func (b *InMemoryBroker) Stats() *BrokerStats {
	// Allocate a new stats struct
	stats := new(BrokerStats)
	stats.DelayedEvals = make(map[string]*models.Evaluation)
	stats.ByScheduler = make(map[string]*SchedulerStats)

	b.l.RLock()
	defer b.l.RUnlock()

	// Copy all the stats
	stats.TotalReady = b.stats.TotalReady
	stats.TotalInflight = b.stats.TotalInflight
	stats.TotalPending = b.stats.TotalPending
	stats.TotalWaiting = b.stats.TotalWaiting
	stats.TotalCancelable = b.stats.TotalCancelable
	for id, eval := range b.stats.DelayedEvals {
		evalCopy := *eval
		stats.DelayedEvals[id] = &evalCopy
	}
	for sched, subStat := range b.stats.ByScheduler {
		subStatCopy := *subStat
		stats.ByScheduler[sched] = &subStatCopy
	}
	return stats
}

// Cancelable retrieves a batch of previously-pending evaluations that are now
// stale and ready to mark for canceling. The eval RPC will call this with a
// batch size set to avoid sending overly large raft messages.
func (b *InMemoryBroker) Cancelable(batchSize int) []*models.Evaluation {
	b.l.Lock()
	defer b.l.Unlock()

	if batchSize > len(b.cancelable) {
		batchSize = len(b.cancelable)
	}

	cancelable := b.cancelable[:batchSize]
	b.cancelable = b.cancelable[batchSize:]

	b.stats.TotalCancelable = len(b.cancelable)
	return cancelable
}

// registerMetrics registers the broker metrics with the meter. This meter collector will periodically
// collect the metrics and send them to the configured metrics backend.
// TODO: evaluate using UpDownCounter instead of collecting our own stats
func (b *InMemoryBroker) registerMetrics() (metric.Registration, error) {
	return orchestrator.Meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		stats := b.Stats()
		o.ObserveInt64(orchestrator.EvalBrokerPending, int64(stats.TotalPending))
		o.ObserveInt64(orchestrator.EvalBrokerWaiting, int64(stats.TotalWaiting))
		o.ObserveInt64(orchestrator.EvalBrokerCancelable, int64(stats.TotalCancelable))
		for sched, schedStats := range stats.ByScheduler {
			attr := orchestrator.EvalTypeAttribute(sched)
			o.ObserveInt64(orchestrator.EvalBrokerReady, int64(schedStats.Ready), attr)
			o.ObserveInt64(orchestrator.EvalBrokerInflight, int64(schedStats.Inflight), attr)
		}
		return nil
	}, orchestrator.EvalBrokerReady, orchestrator.EvalBrokerInflight, orchestrator.EvalBrokerPending,
		orchestrator.EvalBrokerWaiting, orchestrator.EvalBrokerCancelable)
}
