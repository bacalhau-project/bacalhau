package computenode

import (
	"context"
	"fmt"
	"sync"

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

type ComputeNodeConfig struct {
	// this contains things like data locality and per
	// job resource limits
	JobSelectionPolicy JobSelectionPolicy
	// the total amount of CPU and RAM we want to
	// give to running bacalhau jobs
	ResourceLimits resourceusage.ResourceUsageConfig
}

type ComputeNode struct {
	NodeID    string
	Mutex     sync.Mutex
	Transport transport.Transport
	Executors map[executor.EngineType]executor.Executor
	Verifiers map[verifier.VerifierType]verifier.Verifier
	// jobs that are currently running in their executor
	RunningJobs map[string]*executor.Job
	// a FIFO queue of jobs that we selected to run by our JobSelectionPolicy
	// but didn't have enough capacity to run at the time
	// there is a control loop that will attempt to clear this queue
	// it will only bid on jobs it currently has capacity for
	PausedJobs map[string]*executor.Job

	// the config for this compute node
	// things like job selection policy and configured resource limits
	// live here
	Config ComputeNodeConfig

	// how much resources do we have available
	// this either comes from config values or the actual physical resources
	// if the configured values are more than the actual physical resources
	// then we expect an error here
	AvailableResources resourceusage.ResourceUsageData
}

func NewDefaultComputeNodeConfig() ComputeNodeConfig {
	return ComputeNodeConfig{
		JobSelectionPolicy: NewDefaultJobSelectionPolicy(),
		ResourceLimits:     resourceusage.NewDefaultResourceUsageConfig(),
	}
}

// TODO: Clean up function so it's not so long
// nolint:funlen // we get it, it's a big function
func NewComputeNode(
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

	// this is either what the physical CPU / memory values are
	// or the user defined limits from the config (if defined and less than physical amounts)
	availableResources, err := resourceusage.GetSystemResources(config.ResourceLimits)
	if err != nil {
		return nil, err
	}

	computeNode := &ComputeNode{
		NodeID:             nodeID,
		Transport:          t,
		Verifiers:          verifiers,
		Executors:          executors,
		RunningJobs:        map[string]*executor.Job{},
		Config:             config,
		AvailableResources: availableResources,
	}

	t.Subscribe(ctx, func(ctx context.Context, jobEvent *executor.JobEvent, job *executor.Job) {
		var span trace.Span
		switch jobEvent.EventName {
		case executor.JobEventCreated:
			ctx, span = computeNode.newSpanForJob(ctx, job.ID, "JobEventCreated")
			defer span.End()

			// Increment the number of jobs seen by this compute node:
			jobsReceived.With(prometheus.Labels{"node_id": nodeID}).Inc()

			resourceProfile, err := computeNode.getResourceUsageProfile(jobEvent.JobSpec)
			if err != nil {
				log.Error().Msgf("error getting resource profile: %v", err)
				return
			}

			// A new job has arrived - decide if we want to bid on it:
			shouldRun, err := computeNode.SelectJob(ctx,
				JobSelectionPolicyProbeData{
					NodeID:    nodeID,
					JobID:     jobEvent.JobID,
					Resources: resourceProfile,
					Spec:      jobEvent.JobSpec,
				})
			if err != nil {
				log.Error().Msgf("error checking job policy: %v", err)
				return
			}

			if shouldRun {
				// TODO: Why do we have two different kinds of loggers?
				logger.LogJobEvent(logger.JobEvent{
					Node: nodeID,
					Type: "compute_node:bid",
					Job:  job.ID,
				})

				log.Debug().Msgf("compute node %s bidding on: %+v", nodeID,
					jobEvent.JobSpec)

				// TODO: Check result of bid job
				err = t.BidJob(ctx, jobEvent.JobID)
				if err != nil {
					log.Error().Msgf("error bidding on job: %v", err)
				}
				return
			} else {
				log.Debug().Msgf("compute node %s skipped bidding on: %+v",
					nodeID, jobEvent.JobSpec)
			}

		// we have been given the goahead to run the job
		case executor.JobEventBidAccepted:
			// we only care if the accepted bid is for us
			if jobEvent.NodeID != nodeID {
				return
			}

			ctx, span = computeNode.newSpanForJob(ctx,
				job.ID, "JobEventBidAccepted")
			defer span.End()

			// Increment the number of jobs accepted by this compute node:
			jobsAccepted.With(prometheus.Labels{"node_id": nodeID}).Inc()

			log.Debug().Msgf("Bid accepted: Server (id: %s) - Job (id: %s)", nodeID, job.ID)
			logger.LogJobEvent(logger.JobEvent{
				Node: nodeID,
				Type: "compute_node:run",
				Job:  job.ID,
				Data: job,
			})

			resultFolder, err := computeNode.RunJob(ctx, job)
			if err != nil {
				log.Error().Msgf("Error running the job: %s %+v", err, job)
				_ = t.ErrorJob(ctx, job.ID, fmt.Sprintf("Error running the job: %s", err))

				// Increment the number of jobs failed by this compute node:
				jobsFailed.With(prometheus.Labels{"node_id": nodeID}).Inc()

				return
			}

			v, err := computeNode.getVerifier(ctx, job.Spec.Verifier)
			if err != nil {
				log.Error().Msgf("error getting the verifier for the job: %s %+v", err, job)
				_ = t.ErrorJob(ctx, job.ID, fmt.Sprintf("error getting the verifier for the job: %s", err))
				return
			}

			resultValue, err := v.ProcessResultsFolder(
				ctx, job.ID, resultFolder)
			if err != nil {
				log.Error().Msgf("Error verifying results: %s %+v", err, job)
				_ = t.ErrorJob(ctx, job.ID, fmt.Sprintf("Error verifying results: %s", err))
				return
			}

			logger.LogJobEvent(logger.JobEvent{
				Node: nodeID,
				Type: "compute_node:result",
				Job:  job.ID,
				Data: resultValue,
			})

			if err = t.SubmitResult(
				ctx,
				job.ID,
				fmt.Sprintf("Got job result: %s", resultValue),
				resultValue,
			); err != nil {
				log.Error().Msgf("Error submitting result: %s %+v", err, job)
				_ = t.ErrorJob(ctx, job.ID, fmt.Sprintf("Error running the job: %s", err))
				return
			}

			// Increment the number of jobs completed by this compute node:
			jobsCompleted.With(prometheus.Labels{"node_id": nodeID}).Inc()
		}
	})

	return computeNode, nil
}

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

	// decide if we want to take on the job based on
	// our selection policy
	passedJobSelection, err := ApplyJobSelectionPolicy(
		ctx,
		node.Config.JobSelectionPolicy,
		e,
		data,
	)
	if err != nil {
		return false, err
	}
	if !passedJobSelection {
		return false, nil
	}

	// now let's take a look at how many jobs we are currently running (across all of our executors)
	// and decide if we have enough capacity right now to run this

	return true, nil
}

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
func (node *ComputeNode) getResourcesUsing() resourceusage.ResourceUsageData {

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

// what resources are we allowed to use
func (node *ComputeNode) getResourcesTotal() resourceusage.ResourceUsageData {
	return resourceusage.ResourceUsageData{
		CPU:    0,
		Memory: 0,
	}
}

func (node *ComputeNode) getResourceUsageProfile(spec *executor.JobSpec) (resourceusage.ResourceUsageProfile, error) {
	data := resourceusage.ResourceUsageProfile{}
	jobResources, err := resourceusage.ParseResourceUsageConfig(spec.Resources)
	if err != nil {
		return data, err
	}
	return resourceusage.ResourceUsageProfile{
		Job:         jobResources,
		SystemUsing: node.getResourcesUsing(),
		SystemTotal: node.getResourcesTotal(),
	}, nil
}
