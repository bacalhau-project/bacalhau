package compute_node

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type JobSelectionDataLocality int64

const (
	Local    JobSelectionDataLocality = 0
	Anywhere JobSelectionDataLocality = 1
)

// describe the rules for how a compute node selects an incoming job
type JobSelectionPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Locality JobSelectionDataLocality `json:"locality"`
	// should we reject jobs that don't specify any data
	// the default is "accept"
	RejectStatelessJobs bool `json:"reject_stateless_jobs"`
	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHttp string `json:"probe_http,omitempty"`
	ProbeExec string `json:"probe_exec,omitempty"`
}

// the JSON data we send to http or exec probes
type JobSelectionPolicyProbeData struct {
	NodeId string         `json:"node_id"`
	JobId  string         `json:"job_id"`
	Spec   *types.JobSpec `json:"spec"`
}

// generate a default empty job selection policy
func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{}
}

func applyJobSelectionPolicyExecProbe(
	ctx context.Context,
	command string,
	data JobSelectionPolicyProbeData,
) (bool, error) {

	json_data, err := json.Marshal(data)

	if err != nil {
		log.Error().Msgf("Error marshalling job selection policy probe data: %s", err.Error())
		return false, err
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Env = []string{
		"BACALHAU_JOB_SELECTION_PROBE_DATA=" + string(json_data),
	}
	cmd.Stdin = strings.NewReader(string(json_data))
	err = cmd.Run()
	if err != nil {
		// we ignore this error because it might be the script exiting 1 on purpose
		log.Debug().Msgf("We got an error back from a job selection probe exec: %s %s", command, err.Error())
	}

	return cmd.ProcessState.ExitCode() == 0, nil
}

func applyJobSelectionPolicyHttpProbe(
	ctx context.Context,
	url string,
	data JobSelectionPolicyProbeData,
) (bool, error) {

	json_data, err := json.Marshal(data)

	if err != nil {
		log.Error().Msgf("Error marshalling job selection policy probe data: %s", err.Error())
		return false, err
	}

	resp, err := http.Post(url, "application/json",
		bytes.NewBuffer(json_data))

	if err != nil {
		log.Error().Msgf("Error http POST job selection policy probe data: %s %s", url, err.Error())
		return false, err
	}

	return resp.StatusCode == 200, nil
}

func applyJobSelectionPolicyDataSettings(
	ctx context.Context,
	policy JobSelectionPolicy,
	executor executor.Executor,
	job *types.JobSpec,
) (bool, error) {

	// Accept jobs where there are no cids specified
	// if policy.RejectStatelessJobs is set then we reject this job
	if len(job.Inputs) == 0 {
		if policy.RejectStatelessJobs {
			log.Info().Msgf("Found policy of RejectStatelessJobs - rejecting job")
			return false, nil
		} else {
			return true, nil
		}
	}

	// if we have an "anywhere" policy for the data then we accept the job
	if policy.Locality == Anywhere {
		log.Info().Msgf("Found policy of anywhere - accepting job")
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
	data JobSelectionPolicyProbeData,
) (bool, error) {
	if policy.ProbeExec != "" {
		return applyJobSelectionPolicyExecProbe(ctx, policy.ProbeExec, data)
	} else if policy.ProbeHttp != "" {
		return applyJobSelectionPolicyHttpProbe(ctx, policy.ProbeHttp, data)
	} else {
		return applyJobSelectionPolicyDataSettings(ctx, policy, executor, data.Spec)
	}
}
