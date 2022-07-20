package computenode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const DefaultJobCPU = "100m"
const DefaultJobMemory = "100Mb"
const ControlLoopIntervalSeconds = 10

type ComputeNodeConfig struct {
	// this contains things like data locality and per
	// job resource limits
	JobSelectionPolicy JobSelectionPolicy

	// configure the resource capacity we are allowing for
	// this compute node
	CapacityManagerConfig capacitymanager.Config
}

type ComputeNode struct {
	// The ID of this compute node in its configured transport.
	id string

	// The configuration used to create this compute node.
	config ComputeNodeConfig // nolint:gocritic

	controller      *controller.Controller
	executors       map[executor.EngineType]executor.Executor
	verifiers       map[verifier.VerifierType]verifier.Verifier
	capacityManager *capacitymanager.CapacityManager
	componentMu     sync.Mutex
}

func NewDefaultComputeNodeConfig() ComputeNodeConfig {
	return ComputeNodeConfig{
		JobSelectionPolicy: NewDefaultJobSelectionPolicy(),
	}
}

func NewComputeNode(
	cm *system.CleanupManager,
	c *controller.Controller,
	executors map[executor.EngineType]executor.Executor,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	config ComputeNodeConfig, //nolint:gocritic
) (*ComputeNode, error) {
	computeNode, err := constructComputeNode(c, executors, verifiers, config)
	if err != nil {
		return nil, err
	}

	computeNode.subscriptionSetup()
	go computeNode.controlLoopSetup(cm)

	return computeNode, nil
}

// process the arguments and return a valid compoute node
func constructComputeNode(
	c *controller.Controller,
	executors map[executor.EngineType]executor.Executor,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	config ComputeNodeConfig, // nolint:gocritic
) (*ComputeNode, error) {
	// TODO: instrument with trace
	ctx := context.Background()
	nodeID, err := c.HostID(ctx)
	if err != nil {
		return nil, err
	}

	capacityManager, err := capacitymanager.NewCapacityManager(config.CapacityManagerConfig)
	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		id:              nodeID,
		config:          config,
		controller:      c,
		executors:       executors,
		verifiers:       verifiers,
		capacityManager: capacityManager,
	}

	return computeNode, nil
}

/*

  control loops

*/

func (node *ComputeNode) controlLoopSetup(cm *system.CleanupManager) {
	// this won't hurt our throughput becauase we are calling
	// controlLoopBidOnJobs right away as soon as a created event is
	// seen or a job has finished
	ticker := time.NewTicker(time.Second * ControlLoopIntervalSeconds)
	ctx, cancelFunction := context.WithCancel(context.Background())

	cm.RegisterCallback(func() error {
		cancelFunction()
		return nil
	})

	for {
		select {
		case <-ticker.C:
			node.controlLoopBidOnJobs()
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// each control loop we should bid on jobs in our queue
//   * calculate "remaining resources"
//     * this is total - running
//   * loop over each job in selected queue
//     * if there is enough in the remaining then bid
//   * add each bid on job to the "projected resources"
//   * repeat until project resources >= total resources or no more jobs in queue
func (node *ComputeNode) controlLoopBidOnJobs() {
	bidJobIds := node.capacityManager.GetNextItems()
	for _, id := range bidJobIds {
		job, err := node.controller.GetJob(context.Background(), id)
		if err != nil {
			node.capacityManager.Remove(id)
			continue
		}
		err = node.BidOnJob(context.Background(), job.Data)
		if err != nil {
			node.capacityManager.Remove(job.ID)
			continue
		}
		// we did not get an error from the transport
		// so let's assume that our bid is out there
		// now we reserve space on this node for this job
		err = node.capacityManager.MoveToActive(job.ID)
		if err != nil {
			node.capacityManager.Remove(job.ID)
			continue
		}
	}
}

/*

  subscriptions

*/
func (node *ComputeNode) subscriptionSetup() {
	node.controller.Subscribe(func(ctx context.Context, jobEvent executor.JobEvent) {
		job, err := node.controller.GetJob(ctx, jobEvent.JobID)
		if err != nil {
			log.Error().Msgf("could not get job: %s - %s", jobEvent.JobID, err.Error())
			return
		}
		switch jobEvent.EventName {
		case executor.JobEventCreated:
			node.subscriptionEventCreated(ctx, jobEvent, job.Data)
		// we have been given the goahead to run the job
		case executor.JobEventBidAccepted:
			node.subscriptionEventBidAccepted(ctx, jobEvent, job.Data)
		// our bid has not been accepted - let's remove this job from our current queue
		case executor.JobEventBidRejected:
			node.subscriptionEventBidRejected(ctx, jobEvent, job.Data)
		}
	})
}

/*

  subscriptions -> created

*/
func (node *ComputeNode) subscriptionEventCreated(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventCreated")
	defer span.End()

	// Increment the number of jobs seen by this compute node:
	jobsReceived.With(prometheus.Labels{"node_id": node.id}).Inc()

	// A new job has arrived - decide if we want to bid on it:
	selected, processedRequirements, err := node.SelectJob(ctx, JobSelectionPolicyProbeData{
		NodeID: node.id,
		JobID:  jobEvent.JobID,
		Spec:   jobEvent.JobSpec,
	})
	if err != nil {
		log.Error().Msgf("Error checking job policy: %v", err)
		return
	}

	if selected {
		node.capacityManager.AddToBacklog(job.ID, processedRequirements)
		node.controlLoopBidOnJobs()
	}
}

/*

  subscriptions -> bid accepted

*/
func (node *ComputeNode) subscriptionEventBidAccepted(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	var span trace.Span
	// we only care if the accepted bid is for us
	if jobEvent.TargetNodeID != node.id {
		return
	}

	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventBidAccepted")
	defer span.End()

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.With(prometheus.Labels{"node_id": node.id}).Inc()

	log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", node.id, job.ID)
	logger.LogJobEvent(logger.JobEvent{
		Node: node.id,
		Type: "compute_node:run",
		Job:  job.ID,
		Data: job,
	})

	resultFolder, containerRunError := node.RunJob(ctx, job)
	if containerRunError != nil {
		jobsFailed.With(prometheus.Labels{"node_id": node.id}).Inc()
	} else {
		jobsCompleted.With(prometheus.Labels{"node_id": node.id}).Inc()
	}
	if resultFolder == "" {
		errMessage := fmt.Sprintf("Missing results folder for job %s", job.ID)
		if containerRunError != nil {
			errMessage = fmt.Sprintf("RunJob error %s: %s", job.ID, containerRunError)
		}
		log.Error().Msgf(errMessage)
		_ = node.controller.ErrorJob(ctx, job.ID, errMessage, "")
		return
	}

	v, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		errMessage := fmt.Sprintf("Error getting verifier: %s %+v", err, job)
		log.Error().Msgf(errMessage)
		_ = node.controller.ErrorJob(ctx, job.ID, errMessage, "")
		return
	}

	resultValue, err := v.ProcessResultsFolder(ctx, job.ID, resultFolder)
	if err != nil {
		errMessage := fmt.Sprintf("Error verifying results: %s %+v", err, job)
		log.Error().Msg(errMessage)
		_ = node.controller.ErrorJob(ctx, job.ID, errMessage, "")
		return
	}

	if containerRunError == nil {
		logger.LogJobEvent(logger.JobEvent{
			Node: node.id,
			Type: "compute_node:result",
			Job:  job.ID,
			Data: resultValue,
		})
		err = node.controller.CompleteJob(
			ctx,
			job.ID,
			fmt.Sprintf("Got job result: %s", resultValue),
			resultValue,
		)
		if err != nil {
			errMessage := fmt.Sprintf("Error submitting results: %s %+v", err, job)
			log.Error().Msgf(errMessage)
			_ = node.controller.ErrorJob(ctx, job.ID, errMessage, "")
		}
	} else {
		errMessage := fmt.Sprintf("Error running job: %s %+v", containerRunError.Error(), job)
		_ = node.controller.ErrorJob(ctx, job.ID, errMessage, resultValue)
	}
}

/*

  subscriptions -> bid rejected

*/
func (node *ComputeNode) subscriptionEventBidRejected(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	node.capacityManager.Remove(job.ID)
	node.controlLoopBidOnJobs()
}

/*

  job selection

*/
// ask the job selection policy if we would consider running this job
// we return the processed resourceusage.ResourceUsageData for the job
func (node *ComputeNode) SelectJob(ctx context.Context, data JobSelectionPolicyProbeData) (bool, capacitymanager.ResourceUsageData, error) {
	requirements := capacitymanager.ResourceUsageData{}

	// check that we have the executor and it's installed
	e, err := node.getExecutor(ctx, data.Spec.Engine)
	if err != nil {
		return false, requirements, fmt.Errorf("getExecutor: %v", err)
	}

	// check that we have the verifier and it's installed
	_, err = node.getVerifier(ctx, data.Spec.Verifier)
	if err != nil {
		return false, requirements, fmt.Errorf("getVerifier: %v", err)
	}

	// caculate resource requirements for this job
	// this is just parsing strings to ints
	requirements = capacitymanager.ParseResourceUsageConfig(data.Spec.Resources)

	// calculate the disk space we would require if we ran this job
	// this is asking the executor for GetVolumeSize
	diskSpace, err := node.getJobDiskspaceRequirements(ctx, data.Spec)
	if err != nil {
		return false, requirements, fmt.Errorf("error getting job disk space requirements: %v", err)
	}

	// update the job requirements disk space with what we calculated
	requirements.Disk = diskSpace

	withinCapacityLimits, processedRequirements := node.capacityManager.FilterRequirements(requirements)

	if !withinCapacityLimits {
		log.Debug().Msgf("Compute node %s skipped bidding on job because resource requirements were too much: %+v",
			node.id, data.Spec)
		return false, processedRequirements, nil
	}

	// decide if we want to take on the job based on
	// our selection policy
	acceptedByPolicy, err := ApplyJobSelectionPolicy(
		ctx,
		node.config.JobSelectionPolicy,
		e,
		data,
	)

	if err != nil {
		return false, processedRequirements, fmt.Errorf("error selecting job by policy: %v", err)
	}

	if !acceptedByPolicy {
		log.Debug().Msgf("Compute node %s skipped bidding on job because policy did not pass: %+v",
			node.id, data.Spec)
		return false, processedRequirements, nil
	}

	return true, processedRequirements, nil
}

// by bidding on a job - we are moving it from "backlog" to "active"
// in the capacity manager
func (node *ComputeNode) BidOnJob(ctx context.Context, job executor.Job) error {
	// TODO: Why do we have two different kinds of loggers?
	logger.LogJobEvent(logger.JobEvent{
		Node: node.id,
		Type: "compute_node:bid",
		Job:  job.ID,
	})

	log.Debug().Msgf("compute node %s bidding on: %+v", node.id, job.Spec)

	return node.controller.BidJob(ctx, job.ID)
}

/*

  run job

*/
func (node *ComputeNode) RunJob(ctx context.Context, job executor.Job) (string, error) {
	// whatever happens here (either completion or error)
	// we will want to free up the capacity manager from this job
	defer func() {
		node.capacityManager.Remove(job.ID)
		node.controlLoopBidOnJobs()
	}()

	// check that we have the executor to run this job
	e, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return "", err
	}

	return e.RunJob(ctx, job)
}

// nolint:dupl // methods are not duplicates
func (node *ComputeNode) getExecutor(ctx context.Context, typ executor.EngineType) (executor.Executor, error) {
	node.componentMu.Lock()
	defer node.componentMu.Unlock()

	if _, ok := node.executors[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching executor found on this server: %s", typ.String())
	}

	executorEngine := node.executors[typ]
	installed, err := executorEngine.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("executor is not installed: %s", typ.String())
	}

	return executorEngine, nil
}

// nolint:dupl // methods are not duplicates
func (node *ComputeNode) getVerifier(ctx context.Context, typ verifier.VerifierType) (verifier.Verifier, error) {
	node.componentMu.Lock()
	defer node.componentMu.Unlock()

	if _, ok := node.verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	v := node.verifiers[typ]
	installed, err := v.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return v, nil
}

func (node *ComputeNode) newSpanForJob(ctx context.Context, jobID, name string) (context.Context, trace.Span) {
	return system.Span(ctx, "compute_node/compute_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("nodeID", node.id),
			attribute.String("jobID", jobID),
		),
	)
}

func (node *ComputeNode) getJobDiskspaceRequirements(ctx context.Context, spec executor.JobSpec) (uint64, error) {
	e, err := node.getExecutor(context.Background(), spec.Engine)
	if err != nil {
		return 0, err
	}

	var total uint64 = 0

	for _, input := range spec.Inputs {
		volumeSize, err := e.GetVolumeSize(ctx, input)
		if err != nil {
			return 0, err
		}
		total += volumeSize
	}

	return total, nil
}
