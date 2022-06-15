package compute_node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type JobSelectionDataLocality int64

const (
	Local    JobSelectionDataLocality = 0
	Anywhere                          = 1
)

type JobSelectionDataPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Locality JobSelectionDataLocality `json:"locality"`
	// should we reject jobs that don't specify any data
	// the default is "accept"
	RejectStatelessJobs bool `json:"reject_stateless_jobs"`
}

// describe the rules for how a compute node selects an incoming job
type JobSelectionPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Data JobSelectionDataPolicy `json:"data"`

	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHttp string `json:"probe_http,omitempty"`
	ProbeExec string `json:"probe_exec,omitempty"`
}

// the JSON data we send to http or exec probes
type JobSelectionPolicyProbeData struct {
	NodeId string         `json:"node_id"`
	Job    *types.JobSpec `json:"job"`
}

// generate a default empty job selection policy
func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{
		Data: JobSelectionDataPolicy{},
	}
}

func applyJobSelectionPolicyExecProbe(
	ctx context.Context,
	command string,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	return false, nil
}

func applyJobSelectionPolicyHttpProbe(
	ctx context.Context,
	url string,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	return false, nil
}

func applyJobSelectionPolicyDataSettings(
	ctx context.Context,
	policy JobSelectionDataPolicy,
	executor executor.Executor,
	job *types.JobSpec,
) (bool, error) {

	// Accept jobs where there are no cids specified
	// if policy.RejectStatelessJobs is set then we reject this job
	if len(job.Inputs) == 0 {
		return !policy.RejectStatelessJobs, nil
	}

	// if we have an "anywhere" policy for the data then we accept the job
	if policy.Locality == Anywhere {
		return true, nil
	}

	// otherwise we are checking that all of the named inputs in the job
	// are local to us
	foundInputs := 0

	for _, input := range job.Inputs {
		// see if the storage engine reports that we have the resource locally
		hasStorage, err := executor.HasStorage(ctx, input)
		if err != nil {
			log.Error().Msgf("Error checking for storage resource locality: %s", err.Error())
			return false, err
		}
		if hasStorage {
			foundInputs++
		}
	}

	if foundInputs >= len(job.Inputs) {
		log.Info().Msgf("Found %d of %d inputs - accepting job", foundInputs, len(job.Inputs))
		return true, nil
	} else {
		log.Info().Msgf("Found %d of %d inputs - passing on job", foundInputs, len(job.Inputs))
		return false, nil
	}
}

// the compute node "SelectJob" function will call out to this to handle
// applying the policy to the incoming job
// we are also given the executor so we can enquire about data locality
func ApplyJobSelectionPolicy(
	ctx context.Context,
	policy JobSelectionPolicy,
	executor executor.Executor,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	if policy.ProbeExec != "" {
		return applyJobSelectionPolicyExecProbe(ctx, policy.ProbeExec, nodeId, job)
	} else if policy.ProbeHttp != "" {
		return applyJobSelectionPolicyHttpProbe(ctx, policy.ProbeHttp, nodeId, job)
	} else {
		return applyJobSelectionPolicyDataSettings(ctx, policy.Data, executor, job)
	}
}
