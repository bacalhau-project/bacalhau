package compute

import (
	"context"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
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
	ID                     string
	DelegateExecutor       Executor
	Callback               Callback
	RunningCapacityTracker capacity.Tracker
	EnqueuedUsageTracker   capacity.UsageTracker
}

// ExecutorBuffer is a backend.Executor implementation that buffers executions locally until enough capacity is
// available to be able to run them. The buffer accepts a delegate backend.Executor that will be used to run the jobs.
// The buffer is implemented as a FIFO queue, where the order of the executions is determined by the order in which
// they were enqueued. However, an execution with high resource usage requirements might be skipped if there are newer
// jobs with lower resource usage requirements that can be executed immediately. This is done to improve utilization
// of compute nodes, though it might result in starvation and should be re-evaluated in the future.
type ExecutorBuffer struct {
	ID               string
	runningCapacity  capacity.Tracker
	enqueuedCapacity capacity.UsageTracker
	delegateService  Executor
	callback         Callback
	running          map[string]*bufferTask
	queuedTasks      *collections.HashedPriorityQueue[string, *bufferTask]
	mu               sync.Mutex
}

func NewExecutorBuffer(params ExecutorBufferParams) *ExecutorBuffer {
	indexer := func(b *bufferTask) string {
		return b.localExecutionState.Execution.ID
	}

	r := &ExecutorBuffer{
		ID:               params.ID,
		runningCapacity:  params.RunningCapacityTracker,
		enqueuedCapacity: params.EnqueuedUsageTracker,
		delegateService:  params.DelegateExecutor,
		callback:         params.Callback,
		running:          make(map[string]*bufferTask),
		queuedTasks:      collections.NewHashedPriorityQueue[string, *bufferTask](indexer),
	}

	return r
}

// Run enqueues the execution and tries to run it if there is enough capacity.
func (s *ExecutorBuffer) Run(ctx context.Context, localExecutionState store.LocalExecutionState) error {
	var err error

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
				Event: models.EventFromError(EventTopicExecutionPreparing, err),
			})
		}
	}()

	// There is no point in enqueuing a job that requires more than the total capacity of the node. Such jobs should
	// have not reached this backend in the first place, and should have been rejected by the frontend when asked to bid
	if !s.runningCapacity.IsWithinLimits(ctx, *execution.TotalAllocatedResources()) {
		err = bacerrors.New("not enough capacity to run job").WithFailsExecution()
		return err
	}

	if s.queuedTasks.Contains(execution.ID) {
		err = bacerrors.New("execution %s already enqueued", execution.ID)
		return err
	}
	if _, ok := s.running[execution.ID]; ok {
		err = bacerrors.New("execution %s already running", execution.ID)
		return err
	}
	s.enqueuedCapacity.Add(ctx, *execution.TotalAllocatedResources())
	s.queuedTasks.Enqueue(newBufferTask(localExecutionState), int64(execution.Job.Priority))
	s.deque()
	return err
}

// doRun triggers the execution by the delegate backend.Executor and frees up the capacity when the execution is done.
func (s *ExecutorBuffer) doRun(ctx context.Context, task *bufferTask) {
	job := task.localExecutionState.Execution.Job
	ctx = telemetry.AddJobIDToBaggage(ctx, job.ID)
	ctx = telemetry.AddNodeIDToBaggage(ctx, s.ID)
	ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/compute.ExecutorBuffer.Run")
	defer span.End()

	innerCtx := ctx
	if !job.IsLongRunning() {
		timeout := job.Task().Timeouts.GetExecutionTimeout()
		if timeout > 0 {
			var cancel context.CancelFunc
			innerCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	ch := make(chan error)
	go func() {
		ch <- s.delegateService.Run(innerCtx, task.localExecutionState)
	}()

	// no need to check for run errors as they are already handled by the delegate backend.Executor and
	// to the callback.
	<-ch

	s.mu.Lock()
	defer s.mu.Unlock()
	s.runningCapacity.Remove(ctx, *task.localExecutionState.Execution.TotalAllocatedResources())
	delete(s.running, task.localExecutionState.Execution.ID)
	s.deque()
}

// deque tries to run the next execution in the queue if there is enough capacity.
// It is called every time a job is finished or enqueued, where a lock is already held.
// TODO: We loop through the queue every time a job runs or finishes, which is not very efficient.
func (s *ExecutorBuffer) deque() {
	ctx := context.Background()

	// There are at most max matches, so try at most that many times
	max := s.queuedTasks.Len()
	for i := 0; i < max; i++ {
		qItem := s.queuedTasks.DequeueWhere(func(task *bufferTask) bool {
			// If we don't have enough resources to run this task, then we will skip it
			queuedResources := task.localExecutionState.Execution.TotalAllocatedResources()
			allocatedResources := s.runningCapacity.AddIfHasCapacity(ctx, *queuedResources)
			if allocatedResources == nil {
				return false
			}

			// Update the execution to include all the resources that have
			// actually been allocated
			task.localExecutionState.Execution.AllocateResources(
				task.localExecutionState.Execution.Job.Task().Name,
				*allocatedResources,
			)

			// Claim the resources now so that we don't count queued resources
			s.enqueuedCapacity.Remove(ctx, *queuedResources)
			return true
		})

		if qItem == nil {
			// We didn't find anything in the queue that matches our resource availability so we will
			// break out of this look as there is nothing else to find
			break
		}

		task := qItem.Value

		// Move the execution to the running list and remove from the list of enqueued IDs
		// before we actually run the task
		execID := task.localExecutionState.Execution.ID
		s.running[execID] = task

		go s.doRun(logger.ContextWithNodeIDLogger(context.Background(), s.ID), task)
	}
}

func (s *ExecutorBuffer) Cancel(_ context.Context, localExecutionState store.LocalExecutionState) error {
	// TODO: Enqueue cancel tasks
	execution := localExecutionState.Execution
	go func() {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), s.ID)
		ctx = telemetry.AddJobIDToBaggage(ctx, execution.Job.ID)
		ctx = telemetry.AddNodeIDToBaggage(ctx, s.ID)
		ctx, span := telemetry.NewSpan(ctx, telemetry.GetTracer(), "pkg/compute.ExecutorBuffer.Cancel")
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
