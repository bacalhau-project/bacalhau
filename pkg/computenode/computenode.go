package computenode

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const DefaultJobCPU = "100m"
const DefaultJobMemory = "100Mb"
const ControlLoopIntervalMinutes = 10
const DelayBeforeBidMillisecondRange = 100

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
	config ComputeNodeConfig

	controller               *controller.Controller
	shardStateManager		*shardStateMachineManager
	executors                map[executor.EngineType]executor.Executor
	executorsInstalledCache  map[executor.EngineType]bool
	verifiers                map[verifier.VerifierType]verifier.Verifier
	verifiersInstalledCache  map[verifier.VerifierType]bool
	publishers               map[publisher.PublisherType]publisher.Publisher
	publishersInstalledCache map[publisher.PublisherType]bool
	capacityManager          *capacitymanager.CapacityManager
	componentMu              sync.Mutex
	bidMu                    sync.Mutex
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
	publishers map[publisher.PublisherType]publisher.Publisher,
	config ComputeNodeConfig, //nolint:gocritic
) (*ComputeNode, error) {
	computeNode, err := constructComputeNode(c, executors, verifiers, publishers, config)
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
	publishers map[publisher.PublisherType]publisher.Publisher,
	config ComputeNodeConfig,
) (*ComputeNode, error) {
	// TODO: instrument with trace
	ctx := context.Background()
	nodeID, err := c.HostID(ctx)
	if err != nil {
		return nil, err
	}

	shardStateManager, err := NewShardComputeStateMachineManager()
	if err != nil {
		return nil, err
	}

	capacityManager, err := capacitymanager.NewCapacityManager(shardStateManager, config.CapacityManagerConfig)
	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		id:                       nodeID,
		config:                   config,
		controller:               c,
		shardStateManager: 		 shardStateManager,
		executors:                executors,
		executorsInstalledCache:  map[executor.EngineType]bool{},
		verifiers:                verifiers,
		verifiersInstalledCache:  map[verifier.VerifierType]bool{},
		publishers:               publishers,
		publishersInstalledCache: map[publisher.PublisherType]bool{},
		capacityManager:          capacityManager,
	}

	computeNode.componentMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ComputeNode.componentMu",
	})
	computeNode.bidMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "ComputeNode.bidMu",
	})
	
	return computeNode, nil
}

/*

  control loops

*/

func (node *ComputeNode) controlLoopSetup(cm *system.CleanupManager) {
	// this won't hurt our throughput becauase we are calling
	// controlLoopBidOnJobs right away as soon as a created event is
	// seen or a job has finished

	ticker := time.NewTicker(time.Minute * ControlLoopIntervalMinutes)
	ctx, cancelFunction := context.WithCancel(context.Background())

	cm.RegisterCallback(func() error {
		cancelFunction()
		return nil
	})

	for {
		select {
		case <-ticker.C:
			node.controlLoopBidOnJobs("tick")
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// each control loop we should bid on jobs in our queue
//   - calculate "remaining resources"
//   - this is total - running
//   - loop over each job in selected queue
//   - if there is enough in the remaining then bid
//   - add each bid on job to the "projected resources"
//   - repeat until project resources >= total resources or no more jobs in queue
func (node *ComputeNode) controlLoopBidOnJobs(debug string) {
	log.Debug().Msgf("[%s] starting controlLoopBidOnJobs because %s, acq lock", node.id[:8], debug)
	node.bidMu.Lock()
	defer node.bidMu.Unlock()
	bidJobIds := node.capacityManager.GetNextItems()
	
	log.Debug().Msgf("len(bidJobIds)=%d", len(bidJobIds))
	for _, flatID := range bidJobIds {
		jobID, shardIndex, err := capacitymanager.ExplodeShardID(flatID)
		if err != nil {
			log.Error().Msgf("error exploding shard id: %s", err)
			continue
		}
		
		shardState, shardStateFound := node.shardStateManager.Get(flatID)
		if !shardStateFound {
			continue
		}

		if shardState.bidSent {
			// possible race condition where a bid was sent for the shard after 
			// preparing the candidates in GetNextItems()
			log.Trace().Msgf("node %s has already bid on job shard %s %d", node.id, jobID, shardIndex)
			continue
		}

		jobState, err := node.controller.GetJobState(context.Background(), jobID)
		if err != nil {
			shardState.Fail("error getting job state from controller")
			continue
		}
		job, err := node.controller.GetJob(context.Background(), jobID)
		if err != nil {
			shardState.Fail("error getting job instance from controller")
			continue
		}

		hasShardReachedCapacity := jobutils.HasShardReachedCapacity(job, jobState, shardIndex)
		if hasShardReachedCapacity {
			log.Debug().Msgf("node %s: shard %d for job %s has already reached capacity - not bidding", node.id, shardIndex, jobID)
			shardState.Fail("shard has reached capacity")
			continue
		}

		shardState.Bid()
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
		case executor.JobEventResultsAccepted:
			node.subscriptionEventResultsAccepted(ctx, jobEvent, job)
		case executor.JobEventResultsRejected:
			node.subscriptionEventResultsRejected(ctx, jobEvent, job)
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
		"node_id":   node.id,
		"client_id": job.ClientID,
	}).Inc()

	// Decide whether we should even consider bidding on the job, early exit if
	// we're not in the active set for this job, given the hash distances.
	// (This is an optimization to avoid all nodes bidding on a job in large networks).

	// TODO XXX: don't hardcode networkSize, calculate this dynamically from
	// libp2p instead somehow. https://github.com/filecoin-project/bacalhau/issues/512
	jobNodeDistanceDelayMs := CalculateJobNodeDistanceDelay( //nolint:gomnd //nolint:gomnd
		1, node.id, jobEvent.JobID, jobEvent.JobDeal.Concurrency,
	)

	// if delay is too high, just exit immediately.
	if jobNodeDistanceDelayMs > 1000 { //nolint:gomnd
		// drop the job on the floor, :-O
		return
	}
	if jobNodeDistanceDelayMs > 0 {
		log.Debug().Msgf("Waiting %d ms before selecting job %s", jobNodeDistanceDelayMs, jobEvent.JobID)
	}

	time.Sleep(time.Millisecond * time.Duration(jobNodeDistanceDelayMs)) //nolint:gosec

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
			log.Error().Msgf("Error selecting job on host %s: %v", node.id, err)
			return
		}

		// now explode the job into shards and add each shard to the backlog
		shardIndexes := capacitymanager.GenerateShardIndexes(job.ExecutionPlan.TotalShards, processedRequirements)

		// even if an error is returned, some shards might have been partially added to the backlog
		for _, shardIndex := range shardIndexes {
			node.shardStateManager.StartShardStateIfNecessery(job.ID, shardIndex, node, processedRequirements)
		}
		if err != nil {
			log.Error().Msgf("Error adding job to backlog on host %s: %v", node.id, err)
			return
		}
	}
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

func diff(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}

func CalculateJobNodeDistanceDelay(networkSize int, nodeID, jobID string, concurrency int) int {
	// Calculate how long to wait to bid on the job by using a circular hashing
	// style approach: Invent a metric for distance between node ID and job ID.
	// If the node and job ID happen to be close to eachother, such that we'd
	// expect that we are one of the N nodes "closest" to the job, bid
	// instantly. Beyond that, back off an amount "stepped" proportional to how
	// far we are from the job. This should evenly spread the work across the
	// network, and have the property of on average only concurrency many nodes
	// bidding on the job, and other nodes not bothering to bid because they
	// will already have seen bid/bidaccepted messages from the close nodes.
	// This will decrease overall network traffic, improving CPU and memory
	// usage in large clusters.
	nodeHash := hash(nodeID)
	jobHash := hash(jobID)
	// Range: 0 through 4,294,967,295. (4 billion)
	distance := diff(nodeHash, jobHash)
	// scale distance per chunk by concurrency (so that many nodes bid on a job
	// with high concurrency). IOW, divide the space up into this many pieces.
	// If concurrency=3 and network size=3, there'll only be one piece and
	// everyone will bid. If concurrency=1 and network size=1 million, there
	// will be a million slices of the hash space.
	chunk := int((float32(concurrency) / float32(networkSize)) * 4294967295) //nolint:gomnd
	// wait 1 second per chunk distance. So, if we land in exactly the same
	// chunk, bid immediately. If we're one chunk away, wait a bit before
	// bidding. If we're very far away, wait a very long time.
	delay := (distance / chunk) * 1000 //nolint:gomnd
	log.Trace().Msgf(
		"node/job %s/%s, %d/%d, dist=%d, chunk=%d, delay=%d",
		nodeID, jobID, nodeHash, jobHash, distance, chunk, delay,
	)
	return delay
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

	_, span = node.newSpanForJob(ctx, job.ID, "JobEventBidAccepted")
	defer span.End()

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.With(prometheus.Labels{
		"node_id":     node.id,
		"shard_index": strconv.Itoa(jobEvent.ShardIndex),
		"client_id":   job.ClientID,
	}).Inc()

	flatID := capacitymanager.FlattenShardID(jobEvent.JobID, jobEvent.ShardIndex)
	if shardState, ok := node.shardStateManager.Get(flatID); ok {
		shardState.Execute()
	} else {
		log.Error().Msgf("Received bid accepted for unknown shard %s", flatID)
	}
}

/*
subscriptions -> bid rejected
*/
func (node *ComputeNode) subscriptionEventBidRejected(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	// we only care if the rejected bid is for us
	if jobEvent.TargetNodeID != node.id {
		return
	}

	flatID := capacitymanager.FlattenShardID(jobEvent.JobID, jobEvent.ShardIndex)
	if shardState, ok := node.shardStateManager.Get(flatID); ok {
		shardState.BidRejected()
	} else {
		log.Debug().Msgf("Received bid rejected for unknown shard %s", flatID)
	}
}

func (node *ComputeNode) subscriptionEventResultsAccepted(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	// we only care if the rejected bid is for us
	if jobEvent.TargetNodeID != node.id {
		return
	}

	flatID := capacitymanager.FlattenShardID(jobEvent.JobID, jobEvent.ShardIndex)
	if shardState, ok := node.shardStateManager.Get(flatID); ok {
		shardState.Publish()
	} else {
		log.Debug().Msgf("Received results accepted for unknown shard %s", flatID)
	}

	
}

func (node *ComputeNode) subscriptionEventResultsRejected(ctx context.Context, jobEvent executor.JobEvent, job executor.Job) {
	// we only care if the rejected bid is for us
	if jobEvent.TargetNodeID != node.id {
		return
	}
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
func (node *ComputeNode) BidOnJob(ctx context.Context, jobID string, shardIndex int) error {
	log.Debug().Msgf("Compute node %s bidding on: %s %d", node.id, jobID, shardIndex)
	return node.controller.BidJob(ctx, jobID, shardIndex)
}

/*
run job
this is a separate method to RunShard because then we can invoke tests on it directly
*/
func (node *ComputeNode) RunShardExecution(ctx context.Context, job executor.Job, shardIndex int, resultFolder string) error {
	// check that we have the executor to run this job
	e, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return err
	}
	return e.RunShard(ctx, job, shardIndex, resultFolder)
}

func (node *ComputeNode) RunShard(
	ctx context.Context,
	jobID string,
	shardIndex int,
) ([]byte, error) {
	shardProposal := []byte{}
	job, err := node.controller.GetJob(ctx, jobID)
	if err != nil {
		return shardProposal, err
	}

	verifier, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		return shardProposal, err
	}
	resultFolder, err := verifier.GetShardResultPath(ctx, job.ID, shardIndex)
	if err != nil {
		return shardProposal, err
	}

	containerRunError := node.RunShardExecution(ctx, job, shardIndex, resultFolder)
	if containerRunError != nil {
		jobsFailed.With(prometheus.Labels{
			"node_id":     node.id,
			"shard_index": strconv.Itoa(shardIndex),
			"client_id":   job.ClientID,
		}).Inc()
	} else {
		jobsCompleted.With(prometheus.Labels{
			"node_id":     node.id,
			"shard_index": strconv.Itoa(shardIndex),
			"client_id":   job.ClientID,
		}).Inc()
	}

	// if there was an error running the job
	// we don't pass the results off to the verifier
	if containerRunError == nil {
		shardProposal, containerRunError = verifier.GetShardProposal(ctx, job.ID, shardIndex, resultFolder)
	}

	return shardProposal, containerRunError
}

func (node *ComputeNode) PublishShard(
	ctx context.Context,
	jobID string,
	shardIndex int,
) error {
	job, err := node.controller.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	verifier, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		return err
	}
	resultFolder, err := verifier.GetShardResultPath(ctx, job.ID, shardIndex)
	if err != nil {
		return err
	}
	publisher, err := node.getPublisher(ctx, job.Spec.Publisher)
	if err != nil {
		return err
	}
	publishedResult, err := publisher.PublishShardResult(ctx, job.ID, node.id, shardIndex, resultFolder)
	if err != nil {
		return err
	}
	err = node.controller.ShardResultsPublished(ctx, job.ID, shardIndex, publishedResult)
	if err != nil {
		return err
	}
	return nil
}

//nolint:dupl // methods are not duplicates
func (node *ComputeNode) getExecutor(ctx context.Context, typ executor.EngineType) (executor.Executor, error) {
	e := func() *executor.Executor {
		node.componentMu.Lock()
		defer node.componentMu.Unlock()
		if _, ok := node.executors[typ]; !ok {
			return nil
		}
		ee := node.executors[typ]
		return &ee
	}()
	if e == nil {
		return nil, fmt.Errorf(
			"no matching executor found on this server: %s", typ.String(),
		)
	}
	executorEngine := *e

	// cache it being installed so we're not hammering it
	if node.executorsInstalledCache[typ] {
		return executorEngine, nil
	}

	installed, err := executorEngine.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("executor is not installed: %s", typ.String())
	}

	node.executorsInstalledCache[typ] = true

	return executorEngine, nil
}

//nolint:dupl // methods are not duplicates
func (node *ComputeNode) getVerifier(ctx context.Context, typ verifier.VerifierType) (verifier.Verifier, error) {
	v := func() *verifier.Verifier {
		node.componentMu.Lock()
		defer node.componentMu.Unlock()
		if _, ok := node.verifiers[typ]; !ok {
			return nil
		}
		vv := node.verifiers[typ]
		return &vv
	}()

	if v == nil {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}
	verifier := *v

	// cache it being installed so we're not hammering it
	if node.verifiersInstalledCache[typ] {
		return verifier, nil
	}

	installed, err := verifier.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	node.verifiersInstalledCache[typ] = true

	return verifier, nil
}

//nolint:dupl // methods are not duplicates
func (node *ComputeNode) getPublisher(ctx context.Context, typ publisher.PublisherType) (publisher.Publisher, error) {
	p := func() *publisher.Publisher {
		node.componentMu.Lock()
		defer node.componentMu.Unlock()
		if _, ok := node.publishers[typ]; !ok {
			return nil
		}
		pp := node.publishers[typ]
		return &pp
	}()

	if p == nil {
		return nil, fmt.Errorf(
			"no matching publisher found on this server: %s", typ.String())
	}
	publisher := *p

	// cache it being installed so we're not hammering it
	if node.publishersInstalledCache[typ] {
		return publisher, nil
	}

	installed, err := publisher.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	node.publishersInstalledCache[typ] = true

	return publisher, nil
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
