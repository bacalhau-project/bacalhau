package nodes

import (
	"context"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Default configuration values
const (
	defaultBatchDelay       = 15 * time.Second
	defaultMaxBatchSize     = 50
	defaultEventChannelSize = 1000
)

// ReEvaluatorParams defines the configuration for the node re-evaluator
type ReEvaluatorParams struct {
	// JobStore provides access to jobs and executions
	JobStore jobstore.Store

	// BatchDelay is the delay before processing batched node events
	// This helps rate-limit evaluation requests during node churn
	BatchDelay time.Duration

	// MaxBatchSize is the maximum number of events to process in one batch
	MaxBatchSize int

	// EventChannelSize is the size of the event channel buffer
	EventChannelSize int

	// Clock is used for time-based operations (optional, defaults to system clock)
	Clock clock.Clock
}

// ReEvaluator handles automatic job re-evaluation when compute nodes change state.
// It processes node connection events and triggers evaluations for affected jobs:
//
//   - When nodes join/change specs: enqueue evaluations for all daemon jobs (new node might fit)
//     and for batch/service jobs that have executions on that node (compatibility check)
//   - When nodes disappear: enqueue evaluations for all jobs that had active executions on that node
//
// The component implements rate limiting to prevent evaluation floods during node churn.
//
// Usage:
//   - Create a ReEvaluator instance
//   - Start the ReEvaluator
//   - Register the ReEvaluator.HandleNodeConnectionEvent with the NodeManager
//   - The ReEvaluator will automatically process events and trigger job evaluations
type ReEvaluator struct {
	jobStore     jobstore.Store
	batchDelay   time.Duration
	maxBatchSize int
	clock        clock.Clock

	// Event processing
	eventChan chan evaluatorNodeEvent

	// Lifecycle management
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
	running   bool
}

// evaluatorEventType represents the type of node event
type evaluatorEventType int

const (
	evaluatorEventJoin evaluatorEventType = iota
	evaluatorEventLeave
)

// String returns the string representation of the node event type
func (t evaluatorEventType) String() string {
	switch t {
	case evaluatorEventJoin:
		return "join"
	case evaluatorEventLeave:
		return "leave"
	default:
		return "unknown"
	}
}

// evaluatorNodeEvent represents a single node event
type evaluatorNodeEvent struct {
	nodeID    string
	eventType evaluatorEventType
}

// evaluatorEventBatch represents a batch of node events for processing
// Uses a map to ensure only the latest event type per node is kept
type evaluatorEventBatch struct {
	nodeEvents map[string]evaluatorEventType // nodeID -> latest event type
}

// NewReEvaluator creates a new node re-evaluator with the given configuration
func NewReEvaluator(params ReEvaluatorParams) (*ReEvaluator, error) {
	// Set defaults
	if params.BatchDelay == 0 {
		params.BatchDelay = defaultBatchDelay
	}
	if params.MaxBatchSize == 0 {
		params.MaxBatchSize = defaultMaxBatchSize
	}
	if params.EventChannelSize == 0 {
		params.EventChannelSize = defaultEventChannelSize
	}
	if params.Clock == nil {
		params.Clock = clock.New()
	}

	// Validate required parameters
	if err := validate.NotNil(params.JobStore, "job store cannot be nil"); err != nil {
		return nil, err
	}

	re := &ReEvaluator{
		jobStore:     params.JobStore,
		batchDelay:   params.BatchDelay,
		maxBatchSize: params.MaxBatchSize,
		clock:        params.Clock,
		eventChan:    make(chan evaluatorNodeEvent, params.EventChannelSize),
		stopCh:       make(chan struct{}),
	}

	return re, nil
}

// Start initializes the node re-evaluator and begins listening for node events.
// This must be called before the component will process any events.
func (re *ReEvaluator) Start(ctx context.Context) error {
	re.startOnce.Do(func() {
		re.running = true

		// Start the event processing goroutine
		re.wg.Add(1)
		go re.eventProcessingLoop(ctx)

		log.Ctx(ctx).Debug().
			Dur("batchDelay", re.batchDelay).
			Int("maxBatchSize", re.maxBatchSize).
			Int("eventChannelSize", cap(re.eventChan)).
			Msg("Node re-evaluator started")
	})

	return nil
}

// Stop gracefully shuts down the node re-evaluator
func (re *ReEvaluator) Stop(ctx context.Context) error {
	re.stopOnce.Do(func() {
		re.running = false
		close(re.stopCh)

		// Wait for any ongoing evaluation tasks to complete
		done := make(chan struct{})
		go func() {
			re.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
		}

		log.Ctx(ctx).Info().Msg("Node re-evaluator stopped")
	})

	return nil
}

// IsRunning returns whether the re-evaluator is currently active
func (re *ReEvaluator) IsRunning() bool {
	return re.running
}

// HandleNodeConnectionEvent processes a node connection state change event
func (re *ReEvaluator) HandleNodeConnectionEvent(event NodeConnectionEvent) {
	if !re.running {
		return
	}

	select {
	case <-re.stopCh:
		return
	default:
	}

	// Determine event type based on connection state change
	var eventType evaluatorEventType
	if event.Current == models.NodeStates.CONNECTED {
		eventType = evaluatorEventJoin
	} else if event.Previous == models.NodeStates.CONNECTED {
		eventType = evaluatorEventLeave
	} else {
		// State change doesn't involve a connection/disconnection, ignore
		return
	}

	log.Debug().
		Str("nodeID", event.NodeID).
		Str("previous", event.Previous.String()).
		Str("current", event.Current.String()).
		Str("eventType", eventType.String()).
		Msg("Processing node connection event")

	// Submit event to channel (non-blocking)
	nodeEvent := evaluatorNodeEvent{
		nodeID:    event.NodeID,
		eventType: eventType,
	}

	select {
	case re.eventChan <- nodeEvent:
		// Event submitted successfully
	default:
		// Channel is full, log and continue
		log.Warn().
			Str("nodeID", event.NodeID).
			Str("eventType", eventType.String()).
			Msg("Event channel full, dropping node event")
	}
}

// eventProcessingLoop is the main event processing goroutine that batches and processes node events
func (re *ReEvaluator) eventProcessingLoop(ctx context.Context) {
	defer re.wg.Done()

	currentBatch := evaluatorEventBatch{
		nodeEvents: make(map[string]evaluatorEventType),
	}
	timer := re.clock.Timer(re.batchDelay)

	// Reset function to clear the current batch and reset timer
	reset := func() {
		timer.Reset(re.batchDelay)
		currentBatch = evaluatorEventBatch{
			nodeEvents: make(map[string]evaluatorEventType),
		}
	}

	defer func() {
		timer.Stop()
		// Drain any remaining events in the channel
		for {
			select {
			case <-re.eventChan:
				// Discard remaining events
			default:
				return
			}
		}
	}()

	for {
		select {
		case <-re.stopCh:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			log.Debug().Int("batchSize", len(currentBatch.nodeEvents)).Msg("Node re-evaluator processing loop by time")
			// Timer expired, process current batch if it exists
			re.processEventBatch(ctx, currentBatch)
			reset() // Reset batch and timer after processing
		case event := <-re.eventChan:
			// Add/update event in batch (latest event wins)
			currentBatch.nodeEvents[event.nodeID] = event.eventType

			// Check if we should process immediately due to batch size limit
			if len(currentBatch.nodeEvents) >= re.maxBatchSize {
				log.Debug().Int("batchSize", len(currentBatch.nodeEvents)).Msg("Node re-evaluator processing loop by max batch size")
				re.processEventBatch(ctx, currentBatch)
				reset()
			} else {
				// Reset timer to extend batch window
				timer.Reset(re.batchDelay)
			}
		}
	}
}

// processEventBatch processes a single batch of node events
func (re *ReEvaluator) processEventBatch(ctx context.Context, batch evaluatorEventBatch) {
	if len(batch.nodeEvents) == 0 {
		// No events to process, return early
		return
	}

	// Collect all affected jobs from both join and leave events
	jobs := make(map[string]string)

	// Collect all node IDs and separate by event type for logging
	allNodeIDs := make([]string, 0, len(batch.nodeEvents))
	var joinNodeIDs []string
	var leaveNodeIDs []string

	for nodeID, eventType := range batch.nodeEvents {
		allNodeIDs = append(allNodeIDs, nodeID)
		switch eventType {
		case evaluatorEventJoin:
			joinNodeIDs = append(joinNodeIDs, nodeID)
		case evaluatorEventLeave:
			leaveNodeIDs = append(leaveNodeIDs, nodeID)
		}
	}

	// Get executions for all nodes regardless of event type
	executions, err := re.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		NodeIDs:        allNodeIDs,
		InProgressOnly: true,
		IncludeJob:     true, // TODO: add JobType to execution model to avoid loading full job
	})
	if err != nil {
		log.Ctx(ctx).Err(err).
			Strs("nodeIDs", allNodeIDs).
			Msg("Failed to get executions for node evaluation")
	} else {
		for _, execution := range executions {
			jobs[execution.JobID] = execution.Job.Type
		}
	}

	// If any node joins, get all in-progress daemon jobs
	if len(joinNodeIDs) > 0 {
		daemonJobs, err := re.jobStore.GetInProgressJobs(ctx, models.JobTypeDaemon)
		if err != nil {
			log.Ctx(ctx).Err(err).
				Msg("Failed to get in-progress daemon jobs for node join evaluation")
		} else {
			for _, job := range daemonJobs {
				jobs[job.ID] = job.Type
			}
		}
	}

	// Enqueue evaluations for all affected jobs
	for jobID, jobType := range jobs {
		// Use appropriate trigger based on whether there were joins or leaves
		var triggerType string
		if len(joinNodeIDs) > 0 {
			triggerType = models.EvalTriggerNodeJoin
		} else {
			triggerType = models.EvalTriggerNodeLeave
		}
		re.enqueueEvaluationForJob(ctx, jobID, jobType, triggerType)
	}

	log.Debug().
		Int("affectedJobs", len(jobs)).
		Int("totalNodes", len(allNodeIDs)).
		Strs("joinNodes", joinNodeIDs).
		Strs("leaveNodes", leaveNodeIDs).
		Msg("Processed node event batch")
}

// enqueueEvaluationForJob creates and enqueues an evaluation for the given job ID
func (re *ReEvaluator) enqueueEvaluationForJob(ctx context.Context, jobID, jobType, triggerType string) {
	eval := models.NewEvaluation().
		WithJobID(jobID).
		WithType(jobType).
		WithTriggeredBy(triggerType)

	if err := re.jobStore.CreateEvaluation(ctx, *eval); err != nil {
		log.Ctx(ctx).Err(err).
			Str("jobID", jobID).
			Str("evaluationID", eval.ID).
			Str("trigger", triggerType).
			Msg("Failed to create evaluation for node event")
	}
}
