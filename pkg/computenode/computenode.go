package computenode

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
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
	bidMu           sync.Mutex
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
	node.bidMu.Lock()
	defer node.bidMu.Unlock()
	bidJobIds := node.capacityManager.GetNextItems()

	for _, flatId := range bidJobIds {
		jobId, shardIndex, err := capacitymanager.ExplodeShardId(flatId)
		if err != nil {
			node.capacityManager.Remove(flatId)
			continue
		}
		jobLocalEvents, err := node.controller.GetJobLocalEvents(context.Background(), jobId)
		if err != nil {
			node.capacityManager.Remove(flatId)
			continue
		}

		hasAlreadyBid := false

		for _, localEvent := range jobLocalEvents {
			if localEvent.EventName == executor.JobLocalEventBid && localEvent.ShardIndex == shardIndex {
				hasAlreadyBid = true
				break
			}
		}

		if hasAlreadyBid {
			log.Info().Msgf("node %s has already bid on job shard %s %d", node.id, jobId, shardIndex)
			node.capacityManager.Remove(flatId)
			continue
		}

		job, err := node.controller.GetJob(context.Background(), jobId)
		if err != nil {
			node.capacityManager.Remove(flatId)
			continue
		}
		err = node.BidOnJob(context.Background(), job, shardIndex)
		if err != nil {
			node.capacityManager.Remove(flatId)
			continue
		}
		// we did not get an error from the transport
		// so let's assume that our bid is out there
		// now we reserve space on this node for this job
		err = node.capacityManager.MoveToActive(flatId)
		if err != nil {
			node.capacityManager.Remove(flatId)
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
			log.Debug().Msgf("[%s] job created: %s", node.id, job.ID)
			node.subscriptionEventCreated(ctx, jobEvent, job)
		// we have been given the goahead to run the job
		case executor.JobEventBidAccepted:
			node.subscriptionEventBidAccepted(ctx, jobEvent, job)
		// our bid has not been accepted - let's remove this job from our current queue
		case executor.JobEventBidRejected:
			node.subscriptionEventBidRejected(ctx, jobEvent, job)
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
	jobsReceived.With(prometheus.Labels{
		"node_id": node.id,
	}).Inc()

	// A new job has arrived - decide if we want to bid on it:
	selected, processedRequirements, err := node.SelectJob(ctx, JobSelectionPolicyProbeData{
		NodeID:        node.id,
		JobID:         jobEvent.JobID,
		Spec:          jobEvent.JobSpec,
		ExecutionPlan: jobEvent.JobExecutionPlan,
	})
	if err != nil {
		log.Error().Msgf("Error checking job policy: %v", err)
		return
	}

	if selected {
		err = node.controller.SelectJob(ctx, jobEvent.JobID)
		if err != nil {
			log.Info().Msgf("Error selecting job on host %s: %v", node.id, err)
			return
		}

		// now explode the job into shards and add each shard to the backlog
		err = node.capacityManager.AddShardsToBacklog(job.ID, job.ExecutionPlan.TotalShards, processedRequirements)
		if err != nil {
			log.Info().Msgf("Error adding job to backlog on host %s: %v", node.id, err)
			return
		}
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
	jobsAccepted.With(prometheus.Labels{
		"node_id":     node.id,
		"shard_index": strconv.Itoa(jobEvent.ShardIndex),
	}).Inc()

	log.Debug().Msgf("Compute node %s bid accepted on: %s %d", node.id, job.ID, jobEvent.ShardIndex)

	// once we've finished this shard - let's see if we should
	// bid on another shard or if we've finished the job
	defer func() {
		node.capacityManager.Remove(capacitymanager.FlattenShardId(job.ID, jobEvent.ShardIndex))
		node.controlLoopBidOnJobs()
	}()

	results, err := node.RunShard(ctx, job, jobEvent.ShardIndex)
	if err == nil {
		node.controller.CompleteJob(
			ctx,
			job.ID,
			jobEvent.ShardIndex,
			fmt.Sprintf("Got job result: %s", results),
			results,
		)
	} else {
		errMessage := fmt.Sprintf("Error running shard %s %d: %s", job.ID, jobEvent.ShardIndex, err.Error())
		log.Error().Msgf(errMessage)
		_ = node.controller.ErrorJob(
			ctx,
			job.ID,
			jobEvent.ShardIndex,
			errMessage,
			results,
		)
		return
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

	// TODO: think about the fact that each shard might be different sizes
	// this is probably good enough for now
	totalShards := data.ExecutionPlan.TotalShards
	if totalShards == 0 {
		totalShards = 1
	}
	// update the job requirements disk space with what we calculated
	requirements.Disk = diskSpace / uint64(totalShards)

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
		log.Debug().Msgf("Compute node %s skipped bidding on job because policy did not pass: %s",
			node.id, data.JobID)
		return false, processedRequirements, nil
	}

	return true, processedRequirements, nil
}

// by bidding on a job - we are moving it from "backlog" to "active"
// in the capacity manager
func (node *ComputeNode) BidOnJob(ctx context.Context, job executor.Job, shardIndex int) error {
	log.Debug().Msgf("Compute node %s bidding on: %s %d", node.id, job.ID, shardIndex)
	return node.controller.BidJob(ctx, job.ID, shardIndex)
}

/*

  run job

*/
func (node *ComputeNode) ExecuteJobShard(ctx context.Context, job executor.Job, shardIndex int) (string, error) {
	// check that we have the executor to run this job
	e, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return "", err
	}
	return e.RunShard(ctx, job, shardIndex)
}

func (node *ComputeNode) RunShard(
	ctx context.Context,
	job executor.Job,
	shardIndex int,
) (string, error) {
	resultFolder, containerRunError := node.ExecuteJobShard(ctx, job, shardIndex)
	if containerRunError != nil {
		jobsFailed.With(prometheus.Labels{
			"node_id":     node.id,
			"shard_index": strconv.Itoa(shardIndex),
		}).Inc()
	} else {
		jobsCompleted.With(prometheus.Labels{
			"node_id":     node.id,
			"shard_index": strconv.Itoa(shardIndex),
		}).Inc()
	}
	if resultFolder == "" {
		err := fmt.Errorf("Missing results folder for job %s", job.ID)
		if containerRunError != nil {
			err = fmt.Errorf("RunJob error %s: %s", job.ID, containerRunError)
		}
		return "", err
	}
	verifier, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		return "", err
	}
	resultValue, err := verifier.ProcessShardResults(ctx, job.ID, shardIndex, resultFolder)
	if err != nil {
		return "", err
	}
	return resultValue, containerRunError
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
