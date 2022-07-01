package computenode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const DEFAULT_JOB_CPU = "100m"
const DEFAULT_JOB_MEMORY = "100Mb"

type ComputeNodeConfig struct {
	// this contains things like data locality and per
	// job resource limits
	JobSelectionPolicy JobSelectionPolicy
	// the total amount of CPU and RAM we want to
	// give to running bacalhau jobs
	TotalResourceLimit resourceusage.ResourceUsageConfig
	// limit the max CPU / Memory usage for any single job
	JobResourceLimit resourceusage.ResourceUsageConfig
	// if a job does not state how much CPU or Memory is used
	// what values should we assume?
	DefaultJobResourceRequirements resourceusage.ResourceUsageConfig
}

type ComputeNode struct {
	NodeID    string
	Mutex     sync.Mutex
	Transport transport.Transport
	Executors map[executor.EngineType]executor.Executor
	Verifiers map[verifier.VerifierType]verifier.Verifier

	// a FIFO queue of jobs that we selected to run by our JobSelectionPolicy
	// but have not yet had accepted bids on - this is our "backlog"
	SelectedJobQueue []*executor.Job

	// jobs that are currently running in their executor
	// any jobs here will not be present in the selected job queue
	RunningJobs map[string]*executor.Job

	// the config for this compute node
	// things like job selection policy and configured resource limits
	// live here
	Config ComputeNodeConfig

	// both of these are is either what the physical CPU / memory values are
	// or the user defined limits from the config
	// if the user defined limits are more than the actual physical
	// amounts we will get an error
	// if job resource limit is more than total resource limit
	// then we will error (in the case both values are supplied)
	TotalResourceLimit             resourceusage.ResourceUsageData
	JobResourceLimit               resourceusage.ResourceUsageData
	DefaultJobResourceRequirements resourceusage.ResourceUsageData
}

func NewDefaultComputeNodeConfig() ComputeNodeConfig {
	return ComputeNodeConfig{
		JobSelectionPolicy: NewDefaultJobSelectionPolicy(),
	}
}

func NewComputeNode(
	cm *system.CleanupManager,
	t transport.Transport,
	executors map[executor.EngineType]executor.Executor,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	config ComputeNodeConfig,
) (*ComputeNode, error) {
	computeNode, err := constructComputeNode(t, executors, verifiers, config)
	if err != nil {
		return nil, err
	}

	computeNode.subscriptionSetup()
	go computeNode.controlLoopSetup(cm)

	return computeNode, nil
}

// process the arguments and return a valid compoute node
func constructComputeNode(
	t transport.Transport,
	executors map[executor.EngineType]executor.Executor,
	verifiers map[verifier.VerifierType]verifier.Verifier,
	config ComputeNodeConfig,
) (*ComputeNode, error) {

	ctx := context.Background()
	nodeID, err := t.HostID(ctx)
	if err != nil {
		return nil, err
	}

	// assign the default config values
	useConfig := config

	// if we've not been given a default job resource limit
	// then let's use some sensible defaults (which are low on purpose)
	if useConfig.DefaultJobResourceRequirements.CPU == "" {
		useConfig.DefaultJobResourceRequirements.CPU = DEFAULT_JOB_CPU
	}

	if useConfig.DefaultJobResourceRequirements.Memory == "" {
		useConfig.DefaultJobResourceRequirements.Memory = DEFAULT_JOB_MEMORY
	}

	totalResourceLimit, err := resourceusage.GetSystemResources(useConfig.TotalResourceLimit)
	if err != nil {
		return nil, err
	}

	// this is the per job resource limit - i.e. no job can use more than this
	// if no values are given - then we will use the system available resources
	jobResourceLimit := resourceusage.ParseResourceUsageConfig(useConfig.JobResourceLimit)

	// the default value for how much CPU / RAM one job says it needs
	// this is for when a job is submitted with no values for CPU & RAM
	// we will assign these values to it
	defaultJobResourceRequirements := resourceusage.ParseResourceUsageConfig(useConfig.DefaultJobResourceRequirements)

	// if we don't have a limit on job size
	// then let's use the total resources we have on the system
	if jobResourceLimit.CPU <= 0 {
		jobResourceLimit.CPU = totalResourceLimit.CPU
	}

	if jobResourceLimit.Memory <= 0 {
		jobResourceLimit.Memory = totalResourceLimit.Memory
	}

	// we can't have one job that uses more than we have
	if jobResourceLimit.CPU > totalResourceLimit.CPU {
		return nil, fmt.Errorf("job resource limit CPU %f is greater than total system limit %f", jobResourceLimit.CPU, totalResourceLimit.CPU)
	}

	if jobResourceLimit.Memory > totalResourceLimit.Memory {
		return nil, fmt.Errorf("job resource limit memory %d is greater than total system limit %d", jobResourceLimit.Memory, totalResourceLimit.Memory)
	}

	// the default for job requirements can't be more than our job limit
	// or we'll never accept any jobs and so this is classed as a config error
	if defaultJobResourceRequirements.CPU > jobResourceLimit.CPU {
		return nil, fmt.Errorf("default job resource CPU %f is greater than limit %f", defaultJobResourceRequirements.CPU, jobResourceLimit.CPU)
	}

	if defaultJobResourceRequirements.Memory > jobResourceLimit.Memory {
		return nil, fmt.Errorf("default job resource Memory %d is greater than limit %d", defaultJobResourceRequirements.Memory, jobResourceLimit.Memory)
	}

	computeNode := &ComputeNode{
		NodeID:                         nodeID,
		Transport:                      t,
		Verifiers:                      verifiers,
		Executors:                      executors,
		Config:                         useConfig,
		TotalResourceLimit:             totalResourceLimit,
		JobResourceLimit:               jobResourceLimit,
		DefaultJobResourceRequirements: defaultJobResourceRequirements,
		RunningJobs:                    map[string]*executor.Job{},
		SelectedJobQueue:               []*executor.Job{},
	}

	return computeNode, nil
}

/*

  control loops

*/
func (node *ComputeNode) controlLoopSetup(cm *system.CleanupManager) {

	// run our control loop every second
	// TODO: decide how often to run this control loop - perhaps make that configurable?
	ticker := time.NewTicker(time.Second * 1)
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

	runningJobResources := node.getRunningJobResources()
	remainingJobResources := resourceusage.ResourceUsageData{
		CPU:    node.TotalResourceLimit.CPU - runningJobResources.CPU,
		Memory: node.TotalResourceLimit.Memory - runningJobResources.Memory,
	}

	for _, queuedJob := range node.SelectedJobQueue {

		// see if we have enough free resources to run this job
		jobRequirements := node.getJobResourceRequirements(queuedJob.Spec.Resources)

		if resourceusage.CheckResourceRequirements(jobRequirements, remainingJobResources) {
			remainingJobResources.CPU -= jobRequirements.CPU
			remainingJobResources.Memory -= jobRequirements.Memory

			// bid on this job
			node.removeSelectedJob(queuedJob.ID)
			node.BidOnJob(context.Background(), queuedJob)
		}
	}

}

/*

  subscriptions

*/
func (node *ComputeNode) subscriptionSetup() {
	node.Transport.Subscribe(context.Background(), func(ctx context.Context, jobEvent *executor.JobEvent, job *executor.Job) {
		switch jobEvent.EventName {
		case executor.JobEventCreated:
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
func (node *ComputeNode) subscriptionEventCreated(ctx context.Context, jobEvent *executor.JobEvent, job *executor.Job) {
	var span trace.Span
	ctx, span = node.newSpanForJob(ctx, job.ID, "JobEventCreated")
	defer span.End()

	// Increment the number of jobs seen by this compute node:
	jobsReceived.With(prometheus.Labels{"node_id": node.NodeID}).Inc()

	// A new job has arrived - decide if we want to bid on it:
	isJobSelected, err := node.SelectJob(ctx, JobSelectionPolicyProbeData{
		NodeID: node.NodeID,
		JobID:  jobEvent.JobID,
		Spec:   jobEvent.JobSpec,
	})

	if err != nil {
		log.Error().Msgf("error checking job policy: %v", err)
		return
	}

	if isJobSelected {

		// add the job to the queue on selected jobs
		node.addSelectedJob(job)
		node.controlLoopBidOnJobs()

		// err = node.BidOnJob(ctx, job)
		// if err != nil {
		// 	log.Error().Msgf("error bidding on job: %v", err)
		// 	return
		// }
	} else {
		log.Debug().Msgf("compute node %s skipped bidding on: %+v",
			node.NodeID, jobEvent.JobSpec)
	}
}

/*

  subscriptions -> bid accepted

*/
func (node *ComputeNode) subscriptionEventBidAccepted(ctx context.Context, jobEvent *executor.JobEvent, job *executor.Job) {
	var span trace.Span
	// we only care if the accepted bid is for us
	if jobEvent.NodeID != node.NodeID {
		return
	}

	// TODO: what if we have started and finished the job quicker than the libp2p
	// message came back - we need to know "have I already completed this job?"
	_, ok := node.RunningJobs[job.ID]
	if ok {
		log.Debug().Msgf("Already running job so ignore: %s", job.ID)
		return
	}

	ctx, span = node.newSpanForJob(ctx,
		job.ID, "JobEventBidAccepted")
	defer span.End()

	// Increment the number of jobs accepted by this compute node:
	jobsAccepted.With(prometheus.Labels{"node_id": node.NodeID}).Inc()

	log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", node.NodeID, job.ID)
	logger.LogJobEvent(logger.JobEvent{
		Node: node.NodeID,
		Type: "compute_node:run",
		Job:  job.ID,
		Data: job,
	})

	resultFolder, err := node.RunJob(ctx, job)
	if err != nil {
		log.Error().Msgf("Error running the job: %s %+v", err, job)
		_ = node.Transport.ErrorJob(ctx, job.ID, fmt.Sprintf("Error running the job: %s", err))

		// Increment the number of jobs failed by this compute node:
		jobsFailed.With(prometheus.Labels{"node_id": node.NodeID}).Inc()

		return
	}

	v, err := node.getVerifier(ctx, job.Spec.Verifier)
	if err != nil {
		log.Error().Msgf("error getting the verifier for the job: %s %+v", err, job)
		_ = node.Transport.ErrorJob(ctx, job.ID, fmt.Sprintf("error getting the verifier for the job: %s", err))
		return
	}

	resultValue, err := v.ProcessResultsFolder(
		ctx, job.ID, resultFolder)
	if err != nil {
		log.Error().Msgf("Error verifying results: %s %+v", err, job)
		_ = node.Transport.ErrorJob(ctx, job.ID, fmt.Sprintf("Error verifying results: %s", err))
		return
	}

	logger.LogJobEvent(logger.JobEvent{
		Node: node.NodeID,
		Type: "compute_node:result",
		Job:  job.ID,
		Data: resultValue,
	})

	if err = node.Transport.SubmitResult(
		ctx,
		job.ID,
		fmt.Sprintf("Got job result: %s", resultValue),
		resultValue,
	); err != nil {
		log.Error().Msgf("Error submitting result: %s %+v", err, job)
		_ = node.Transport.ErrorJob(ctx, job.ID, fmt.Sprintf("Error running the job: %s", err))
		return
	}

	// Increment the number of jobs completed by this compute node:
	jobsCompleted.With(prometheus.Labels{"node_id": node.NodeID}).Inc()
}

/*

  subscriptions -> bid rejected

*/
func (node *ComputeNode) subscriptionEventBidRejected(ctx context.Context, jobEvent *executor.JobEvent, job *executor.Job) {
	node.removeSelectedJob(job.ID)
}

/*

  job selection

*/
// ask the job selection policy if we would consider running this job
func (node *ComputeNode) SelectJob(ctx context.Context, data JobSelectionPolicyProbeData) (bool, error) {
	// check that we have the executor and it's installed
	e, err := node.getExecutor(ctx, data.Spec.Engine)
	if err != nil {
		return false, err
	}

	// check that we have the verifier and it's installed
	_, err = node.getVerifier(ctx, data.Spec.Verifier)
	if err != nil {
		return false, err
	}

	// get the resource requirements for the job
	// this takes into accounts the defaults if the job itself didn't have any requirements
	jobResourceRequirements := node.getJobResourceRequirements(data.Spec.Resources)

	// reject a job that would use more CPU than we would allow
	jobPassesResourceCheck := resourceusage.CheckResourceRequirements(jobResourceRequirements, node.JobResourceLimit)

	if !jobPassesResourceCheck {
		log.Info().Msgf("Job is more than allowed resource usage - rejecting job: job: %+v, limit: %+v", jobResourceRequirements, node.JobResourceLimit)
		return false, nil
	}

	// decide if we want to take on the job based on
	// our selection policy
	return ApplyJobSelectionPolicy(
		ctx,
		node.Config.JobSelectionPolicy,
		e,
		data,
	)
}

func (node *ComputeNode) BidOnJob(ctx context.Context, job *executor.Job) error {
	// TODO: Why do we have two different kinds of loggers?
	logger.LogJobEvent(logger.JobEvent{
		Node: node.NodeID,
		Type: "compute_node:bid",
		Job:  job.ID,
	})

	log.Debug().Msgf("compute node %s bidding on: %+v", node.NodeID, job.Spec)

	// TODO: Check result of bid job
	return node.Transport.BidJob(ctx, job.ID)
}

/*

  run job

*/
func (node *ComputeNode) RunJob(ctx context.Context, job *executor.Job) (string, error) {
	// check that we have the executor to run this job
	e, err := node.getExecutor(ctx, job.Spec.Engine)
	if err != nil {
		return "", err
	}

	node.addRunningJob(job)
	result, err := e.RunJob(ctx, job)
	node.removeRunningJob(job)

	if err != nil {
		return "", err
	}

	return result, nil
}

// nolint:dupl // methods are not duplicates
func (node *ComputeNode) getExecutor(ctx context.Context, typ executor.EngineType) (executor.Executor, error) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Executors[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching executor found on this server: %s", typ.String())
	}

	executorEngine := node.Executors[typ]
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
	node.Mutex.Lock()
	defer node.Mutex.Unlock()

	if _, ok := node.Verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	v := node.Verifiers[typ]
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
			attribute.String("nodeID", node.NodeID),
			attribute.String("jobID", jobID),
		),
	)
}

var selectedJobMutex sync.Mutex

func (node *ComputeNode) addSelectedJob(job *executor.Job) {
	selectedJobMutex.Lock()
	defer selectedJobMutex.Unlock()
	node.SelectedJobQueue = append(node.SelectedJobQueue, job)
}

func (node *ComputeNode) removeSelectedJob(id string) {
	selectedJobMutex.Lock()
	defer selectedJobMutex.Unlock()
	newArr := []*executor.Job{}
	for _, j := range node.SelectedJobQueue {
		if j.ID != id {
			newArr = append(newArr, j)
		}
	}
	node.SelectedJobQueue = newArr
}

var runningJobMutex sync.Mutex

func (node *ComputeNode) addRunningJob(job *executor.Job) {
	runningJobMutex.Lock()
	defer runningJobMutex.Unlock()
	node.RunningJobs[job.ID] = job
}

func (node *ComputeNode) removeRunningJob(job *executor.Job) {
	runningJobMutex.Lock()
	defer runningJobMutex.Unlock()
	delete(node.RunningJobs, job.ID)
}

// add up all the resources being used by all the jobs currently running
func (node *ComputeNode) getRunningJobResources() resourceusage.ResourceUsageData {

	var cpu float64
	var memory uint64

	runningJobMutex.Lock()
	defer runningJobMutex.Unlock()

	for _, job := range node.RunningJobs {
		cpu += resourceusage.ConvertCpuString(job.Spec.Resources.CPU)
		memory += resourceusage.ConvertMemoryString(job.Spec.Resources.Memory)
	}

	return resourceusage.ResourceUsageData{
		CPU:    cpu,
		Memory: memory,
	}
}

// get the limits for a single job
// either using it's configured limits or the compute node default job limits
func (node *ComputeNode) getJobResourceRequirements(jobRequirements resourceusage.ResourceUsageConfig) resourceusage.ResourceUsageData {
	data := resourceusage.ParseResourceUsageConfig(jobRequirements)
	if data.CPU <= 0 {
		data.CPU = node.DefaultJobResourceRequirements.CPU
	}
	if data.Memory <= 0 {
		data.Memory = node.DefaultJobResourceRequirements.Memory
	}
	return data
}
