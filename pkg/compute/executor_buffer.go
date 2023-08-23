package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
)

type bufferTask struct {
	localExecutionState store.LocalExecutionState
	enqueuedAt          time.Time
}

func newBufferTask(execution store.LocalExecutionState) *bufferTask {
	return &bufferTask{
		localExecutionState: execution,
		enqueuedAt:          time.Now(),
	}
}

type ExecutorBufferParams struct {
	ID                         string
	DelegateExecutor           Executor
	Callback                   Callback
	RunningCapacityTracker     capacity.Tracker
	EnqueuedCapacityTracker    capacity.Tracker
	DefaultJobExecutionTimeout time.Duration
	BackoffDuration            time.Duration
}

// ExecutorBuffer is a backend.Executor implementation that buffers executions locally until enough capacity is
// available to be able to run them. The buffer accepts a delegate backend.Executor that will be used to run the jobs.
// The buffer is implemented as a FIFO queue, where the order of the executions is determined by the order in which
// they were enqueued. However, an execution with high resource usage requirements might be skipped if there are newer
// jobs with lower resource usage requirements that can be executed immediately. This is done to improve utilization
// of compute nodes, though it might result in starvation and should be re-evaluated in the future.
type ExecutorBuffer struct {
	ID                         string
	runningCapacity            capacity.Tracker
	enqueuedCapacity           capacity.Tracker
	delegateService            Executor
	callback                   Callback
	running                    map[string]*bufferTask
	queuedTasks                *collections.PriorityQueue[*bufferTask]
	enqueuedIDs                map[string]struct{}
	defaultJobExecutionTimeout time.Duration
	backoffDuration            time.Duration
	backoffUntil               time.Time
	mu                         sync.Mutex
}

func NewExecutorBuffer(params ExecutorBufferParams) *ExecutorBuffer {
	r := &ExecutorBuffer{
		ID:                         params.ID,
		runningCapacity:            params.RunningCapacityTracker,
		enqueuedCapacity:           params.EnqueuedCapacityTracker,
		delegateService:            params.DelegateExecutor,
		callback:                   params.Callback,
		running:                    make(map[string]*bufferTask),
		queuedTasks:                collections.NewPriorityQueue[*bufferTask](),
		enqueuedIDs:                make(map[string]struct{}),
		defaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,
		backoffDuration:            params.BackoffDuration,
	}

	r.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ExecutorBuffer.mu",
	})

	return r
}

// Run enqueues the execution and tries to run it if there is enough capacity.
func (s *ExecutorBuffer) Run(ctx context.Context, localExecutionState store.LocalExecutionState) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution := localExecutionState.Execution

	defer func() {
		if err != nil {
			s.callback.OnComputeFailure(ctx, ComputeError{
				ExecutionMetadata: NewExecutionMetadata(execution),
				RoutingMetadata: RoutingMetadata{
					SourcePeerID: s.ID,
					TargetPeerID: localExecutionState.RequesterNodeID,
				},
				Err: err.Error(),
			})
		}
	}()

	// There is no point in enqueuing a job that requires more than the total capacity of the node. Such jobs should
	// have not reached this backend in the first place, and should have been rejected by the frontend when asked to bid
	if !s.runningCapacity.IsWithinLimits(ctx, *execution.TotalAllocatedResources()) {
		err = fmt.Errorf("not enough capacity to run job")
		return
	}
	if _, ok := s.enqueuedIDs[execution.ID]; ok {
		err = fmt.Errorf("execution %s already enqueued", execution.ID)
		return
	}
	if _, ok := s.running[execution.ID]; ok {
		err = fmt.Errorf("execution %s already running", execution.ID)
		return
	}
	if !s.enqueuedCapacity.AddIfHasCapacity(ctx, *execution.TotalAllocatedResources()) {
		err = fmt.Errorf("not enough capacity to enqueue job")
		return
	}

	s.queuedTasks.Enqueue(newBufferTask(localExecutionState), execution.Job.Priority)
	s.enqueuedIDs[execution.ID] = struct{}{}
	s.deque()
	return err
}

// doRun triggers the execution by the delegate backend.Executor and frees up the capacity when the execution is done.
func (s *ExecutorBuffer) doRun(ctx context.Context, task *bufferTask) {
	ctx = system.AddJobIDToBaggage(ctx, task.localExecutionState.Execution.Job.ID)
	ctx = system.AddNodeIDToBaggage(ctx, s.ID)
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.ExecutorBuffer.Run")
	defer span.End()

	timeout := task.localExecutionState.Execution.Job.Task().Timeouts.GetExecutionTimeout()
	if timeout == 0 {
		timeout = s.defaultJobExecutionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan error)
	go func() {
		ch <- s.delegateService.Run(ctx, task.localExecutionState)
	}()

	select {
	case <-ctx.Done():
		log.Ctx(ctx).Info().Str("ID", task.localExecutionState.Execution.ID).Dur("Timeout", timeout).Msg("Execution timed out")
		s.callback.OnCancelComplete(ctx, CancelResult{
			ExecutionMetadata: NewExecutionMetadata(task.localExecutionState.Execution),
			RoutingMetadata: RoutingMetadata{
				SourcePeerID: s.ID,
				TargetPeerID: task.localExecutionState.RequesterNodeID,
			},
		})
	case <-ch:
		// no need to check for run errors as they are already handled by the delegate backend.Executor and
		// to the callback.
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.runningCapacity.Remove(ctx, *task.localExecutionState.Execution.TotalAllocatedResources())
	delete(s.running, task.localExecutionState.Execution.ID)
	s.deque()
}

// deque tries to run the next execution in the queue if there is enough capacity.
// It is called every time a job is finished or enqueued, where a lock is already held.
func (s *ExecutorBuffer) deque() {
	// If last attempt was very recent, and we still have jobs running,
	// then we need to wait until backoffDuration has passed
	if len(s.running) != 0 && time.Now().Before(s.backoffUntil) {
		return
	}
	ctx := context.Background()

	holdingQueue := collections.NewPriorityQueue[*bufferTask]()

	// We know there are len(s.enqueuedList) items to process, so let's dequeue at most
	// that many entries
	max := len(s.enqueuedIDs)
	for i := 0; i < max; i++ {
		task, prio, err := s.queuedTasks.Dequeue()
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("executor_buffer failed to obtain task from queue")
			break
		}

		execID := task.localExecutionState.Execution.ID

		// We maintain the enqueued execution state as a priority queue, while allowing to skip over jobs
		// that require more resources than the current capacity. This is done to improve utilization of
		// compute nodes, though it might result in starvation and should be re-evaluated in the future.
		if s.runningCapacity.AddIfHasCapacity(ctx, *task.localExecutionState.Execution.TotalAllocatedResources()) {
			s.enqueuedCapacity.Remove(ctx, *task.localExecutionState.Execution.TotalAllocatedResources())

			delete(s.enqueuedIDs, execID)
			s.running[execID] = task

			go s.doRun(logger.ContextWithNodeIDLogger(context.Background(), s.ID), task)
		} else {
			// We will need to re-enqueue this item as we don't have resources for now
			holdingQueue.Enqueue(task, prio)
		}
	}

	// Merge the holding queue back into the queued tasks
	s.queuedTasks.Merge(holdingQueue)
	s.backoffUntil = time.Now().Add(s.backoffDuration)
}

func (s *ExecutorBuffer) Cancel(_ context.Context, localExecutionState store.LocalExecutionState) error {
	// TODO: Enqueue cancel tasks
	execution := localExecutionState.Execution
	go func() {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), s.ID)
		ctx = system.AddJobIDToBaggage(ctx, execution.Job.ID)
		ctx = system.AddNodeIDToBaggage(ctx, s.ID)
		ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.ExecutorBuffer.Cancel")
		defer span.End()

		err := s.delegateService.Cancel(ctx, localExecutionState)
		if err == nil {
			s.mu.Lock()
			defer s.mu.Unlock()

			delete(s.running, execution.ID)
		}
	}()
	return nil
}

// RunningExecutions return list of running executions
func (s *ExecutorBuffer) RunningExecutions() []store.LocalExecutionState {
	return s.mapValues(s.running)
}

// EnqueuedExecutionsCount return number of items enqueued
func (s *ExecutorBuffer) EnqueuedExecutionsCount() int {
	return s.queuedTasks.Len()
}

func (s *ExecutorBuffer) mapValues(m map[string]*bufferTask) []store.LocalExecutionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	executions := make([]store.LocalExecutionState, 0, len(m))
	for _, v := range m {
		executions = append(executions, v.localExecutionState)
	}
	return executions
}

// compile-time interface check
var _ Executor = (*ExecutorBuffer)(nil)
