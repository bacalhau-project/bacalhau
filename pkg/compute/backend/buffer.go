package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	sync "github.com/lukemarsden/golang-mutex-tracer"
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

type ServiceBufferParams struct {
	DelegateService            Service
	Callback                   Callback
	RunningCapacityTracker     capacity.Tracker
	DefaultJobExecutionTimeout time.Duration
	BackoffDuration            time.Duration
}

type ServiceBuffer struct {
	runningCapacity            capacity.Tracker
	delegateService            Service
	callback                   Callback
	running                    map[string]*bufferTask
	enqueued                   map[string]*bufferTask
	enqueuedList               []string
	defaultJobExecutionTimeout time.Duration
	backoffDuration            time.Duration
	backoffUntil               time.Time
	mu                         sync.Mutex
}

func NewServiceBuffer(params ServiceBufferParams) *ServiceBuffer {
	r := &ServiceBuffer{
		runningCapacity:            params.RunningCapacityTracker,
		delegateService:            params.DelegateService,
		callback:                   params.Callback,
		running:                    make(map[string]*bufferTask),
		enqueued:                   make(map[string]*bufferTask),
		enqueuedList:               make([]string, 0),
		defaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,
		backoffDuration:            params.BackoffDuration,
	}

	r.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ServiceBuffer.mu",
	})

	return r
}

func (s *ServiceBuffer) Run(ctx context.Context, execution store.Execution) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		if err != nil {
			s.callback.OnRunFailure(ctx, execution.ID, err)
		}
	}()

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

	s.enqueued[execution.ID] = newBufferTask(execution)
	s.enqueuedList = append(s.enqueuedList, execution.ID)
	s.deque()
	return
}

func (s *ServiceBuffer) doRun(ctx context.Context, task *bufferTask) {
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
		s.callback.OnRunFailure(ctx, task.execution.ID, ctx.Err())
	case runError := <-ch:
		if runError != nil {
			s.callback.OnRunFailure(ctx, task.execution.ID, runError)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.runningCapacity.Remove(ctx, task.execution.ResourceUsage)
	delete(s.running, task.execution.ID)
	s.deque()
}

func (s *ServiceBuffer) deque() {
	// If last attempt was very recent, and we still have jobs running,
	// then we need to wait until backoffDuration has passed
	if len(s.running) != 0 && time.Now().Before(s.backoffUntil) {
		return
	}
	ctx := context.Background()

	// Since we want to keep the list of shard state machines ordered by their creation time,
	// and since shards can complete at any time, we need to remove completed shards
	// from the list without impacting the order of the remaining shards, and without
	// having to copy things around.
	remainingEnqueuedList := make([]string, 0, len(s.enqueuedList))

	for _, executionID := range s.enqueuedList {
		task := s.enqueued[executionID]

		if s.runningCapacity.AddIfHasCapacity(ctx, task.execution.ResourceUsage) {
			delete(s.enqueued, executionID)
			s.running[executionID] = task
			go s.doRun(ctx, task)
		} else {
			remainingEnqueuedList = append(remainingEnqueuedList, executionID)
		}
	}
	s.enqueuedList = remainingEnqueuedList
	s.backoffUntil = time.Now().Add(s.backoffDuration)
}

func (s *ServiceBuffer) Publish(ctx context.Context, execution store.Execution) error {
	return s.delegateService.Publish(ctx, execution)
}

func (s *ServiceBuffer) Cancel(ctx context.Context, execution store.Execution) error {
	return s.delegateService.Cancel(ctx, execution)
}

// RunningExecutions return list of running executions
func (s *ServiceBuffer) RunningExecutions() []store.Execution {
	return s.mapValues(s.running)
}

// EnqueuedExecutions return list of enqueued executions
func (s *ServiceBuffer) EnqueuedExecutions() []store.Execution {
	return s.mapValues(s.enqueued)
}

func (s *ServiceBuffer) mapValues(m map[string]*bufferTask) []store.Execution {
	s.mu.Lock()
	defer s.mu.Unlock()
	executions := make([]store.Execution, 0, len(m))
	for _, v := range m {
		executions = append(executions, v.execution)
	}
	return executions
}

// compile-time interface check
var _ Service = (*ServiceBuffer)(nil)
