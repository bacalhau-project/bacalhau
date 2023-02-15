package compute

import (
	"context"
	"fmt"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type bufferTask struct {
	execution  store.Execution
	enqueuedAt time.Time
}

func newBufferTask(execution store.Execution) *bufferTask {
	return &bufferTask{
		execution:  execution,
		enqueuedAt: time.Now(),
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
	enqueued                   map[string]*bufferTask
	enqueuedList               []string
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
		enqueued:                   make(map[string]*bufferTask),
		enqueuedList:               make([]string, 0),
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
func (s *ExecutorBuffer) Run(ctx context.Context, execution store.Execution) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		if err != nil {
			s.callback.OnComputeFailure(ctx, ComputeError{
				ExecutionMetadata: NewExecutionMetadata(execution),
				RoutingMetadata: RoutingMetadata{
					SourcePeerID: s.ID,
					TargetPeerID: execution.RequesterNodeID,
				},
				Err: err.Error(),
			})
		}
	}()

	// There is no point in enqueuing a job that requires more than the total capacity of the node. Such jobs should
	// have not reached this backend in the first place, and should have been rejected by the frontend when asked to bid
	if !s.runningCapacity.IsWithinLimits(ctx, execution.ResourceUsage) {
		err = fmt.Errorf("not enough capacity to run job")
		return
	}
	if _, ok := s.enqueued[execution.ID]; ok {
		err = fmt.Errorf("execution %s already enqueued", execution.ID)
		return
	}
	if _, ok := s.running[execution.ID]; ok {
		err = fmt.Errorf("execution %s already running", execution.ID)
		return
	}
	if !s.enqueuedCapacity.AddIfHasCapacity(ctx, execution.ResourceUsage) {
		err = fmt.Errorf("not enough capacity to enqueue job")
		return
	}

	s.enqueued[execution.ID] = newBufferTask(execution)
	s.enqueuedList = append(s.enqueuedList, execution.ID)
	s.deque()
	return err
}

// doRun triggers the execution by the delegate backend.Executor and frees up the capacity when the execution is done.
func (s *ExecutorBuffer) doRun(ctx context.Context, task *bufferTask) {
	ctx = system.AddJobIDToBaggage(ctx, task.execution.Shard.Job.Metadata.ID)
	ctx = system.AddNodeIDToBaggage(ctx, s.ID)
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.ExecutorBuffer.Run")
	defer span.End()

	timeout := task.execution.Shard.Job.Spec.GetTimeout()
	if timeout == 0 {
		timeout = s.defaultJobExecutionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan error)
	go func() {
		ch <- s.delegateService.Run(ctx, task.execution)
	}()

	select {
	case <-ctx.Done():
		s.callback.OnComputeFailure(ctx, ComputeError{
			ExecutionMetadata: NewExecutionMetadata(task.execution),
			RoutingMetadata: RoutingMetadata{
				SourcePeerID: s.ID,
				TargetPeerID: task.execution.RequesterNodeID,
			},
			Err: fmt.Sprintf("execution timed out after %s", timeout),
		})
	case <-ch:
		// no need to check for run errors as they are already handled by the delegate backend.Executor and
		// to the callback.
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.runningCapacity.Remove(ctx, task.execution.ResourceUsage)
	delete(s.running, task.execution.ID)
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

	// We are maintain the order of enqueued executions treat it as a FIFO queue, while allowing to skip over jobs
	// that require more resources than the current capacity. This is done to improve utilization of compute nodes,
	// though it might result in starvation and should be re-evaluated in the future.
	remainingEnqueuedList := make([]string, 0, len(s.enqueuedList))

	for _, executionID := range s.enqueuedList {
		task := s.enqueued[executionID]

		if s.runningCapacity.AddIfHasCapacity(ctx, task.execution.ResourceUsage) {
			s.enqueuedCapacity.Remove(ctx, task.execution.ResourceUsage)
			delete(s.enqueued, executionID)
			s.running[executionID] = task
			go s.doRun(logger.ContextWithNodeIDLogger(context.Background(), s.ID), task)
		} else {
			remainingEnqueuedList = append(remainingEnqueuedList, executionID)
		}
	}
	s.enqueuedList = remainingEnqueuedList
	s.backoffUntil = time.Now().Add(s.backoffDuration)
}

func (s *ExecutorBuffer) Publish(_ context.Context, execution store.Execution) error {
	// TODO: Enqueue publish tasks
	go func() {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), s.ID)
		ctx = system.AddJobIDToBaggage(ctx, execution.Shard.Job.Metadata.ID)
		ctx = system.AddNodeIDToBaggage(ctx, s.ID)
		ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.ExecutorBuffer.Publish")
		defer span.End()
		_ = s.delegateService.Publish(ctx, execution)
	}()
	return nil
}

func (s *ExecutorBuffer) Cancel(_ context.Context, execution store.Execution) error {
	// TODO: Enqueue cancel tasks
	go func() {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), s.ID)
		ctx = system.AddJobIDToBaggage(ctx, execution.Shard.Job.Metadata.ID)
		ctx = system.AddNodeIDToBaggage(ctx, s.ID)
		ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/compute.ExecutorBuffer.Cancel")
		defer span.End()
		_ = s.delegateService.Cancel(ctx, execution)
	}()
	return nil
}

// RunningExecutions return list of running executions
func (s *ExecutorBuffer) RunningExecutions() []store.Execution {
	return s.mapValues(s.running)
}

// EnqueuedExecutions return list of enqueued executions
func (s *ExecutorBuffer) EnqueuedExecutions() []store.Execution {
	return s.mapValues(s.enqueued)
}

func (s *ExecutorBuffer) mapValues(m map[string]*bufferTask) []store.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	executions := make([]store.Execution, 0, len(m))
	for _, v := range m {
		executions = append(executions, v.execution)
	}
	return executions
}

// compile-time interface check
var _ Executor = (*ExecutorBuffer)(nil)
