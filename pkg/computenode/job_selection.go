package computenode

import (
	"bytes"
	"context"
	"net/http"
	"os/exec"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
	ProbeHTTP string `json:"probe_http,omitempty"`
	ProbeExec string `json:"probe_exec,omitempty"`
}

// the JSON data we send to http or exec probes
type JobSelectionPolicyProbeData struct {
	NodeID        string                 `json:"node_id"`
	JobID         string                 `json:"job_id"`
	Spec          model.Spec             `json:"spec"`
	ExecutionPlan model.JobExecutionPlan `json:"execution_plan"`
}

// generate a default empty job selection policy
func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{}
}

//nolint:unparam // will fix
func applyJobSelectionPolicyExecProbe(
	ctx context.Context,
	command string,
	data JobSelectionPolicyProbeData, //nolint:gocritic
) (bool, error) {
	// TODO: Use context to trace exec call

	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("error marshaling job selection policy probe data: %s", err.Error())
		return false, err
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Env = []string{
		"BACALHAU_JOB_SELECTION_PROBE_DATA=" + string(jsonData),
	}
	cmd.Stdin = strings.NewReader(string(jsonData))
	err = cmd.Run()
	if err != nil {
		// we ignore this error because it might be the script exiting 1 on purpose
		log.Ctx(ctx).Debug().Msgf("We got an error back from a job selection probe exec: %s %s", command, err.Error())
	}

	return cmd.ProcessState.ExitCode() == 0, nil
}

func applyJobSelectionPolicyHTTPProbe(ctx context.Context, url string, data JobSelectionPolicyProbeData) (bool, error) {
	jsonData, err := model.JSONMarshalWithMax(data)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("error marshaling job selection policy probe data: %s", err.Error())
		return false, err
	}

	body := bytes.NewBuffer(jsonData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		log.Ctx(ctx).Error().Msgf("could not create http request with context: %s", url)
	}
	resp, err := http.DefaultClient.Do(req)
	resp.Body.Close()

	if err != nil {
		log.Ctx(ctx).Error().Msgf("error http POST job selection policy probe data: %s %s", url, err.Error())
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

func applyJobSelectionPolicySettings(
	ctx context.Context,
	policy JobSelectionPolicy,
	e executor.Executor,
	job model.Spec,
) (bool, error) {
	// Accept jobs where there are no cids specified
	// if policy.RejectStatelessJobs is set then we reject this job
	if len(job.Inputs) == 0 {
		if policy.RejectStatelessJobs {
			log.Ctx(ctx).Trace().Msgf("Found policy of RejectStatelessJobs - rejecting job")
			return false, nil
		} else {
			return true, nil
		}
	}

	// if we have an "anywhere" policy for the data then we accept the job
	if policy.Locality == Anywhere {
		log.Ctx(ctx).Trace().Msgf("Found policy of anywhere - accepting job")
		return true, nil
	}

	// otherwise we are checking that all of the named inputs in the job
	// are local to us
	foundInputs := 0

	for _, input := range job.Inputs {
		// see if the storage engine reports that we have the resource locally
		hasStorage, err := e.HasStorageLocally(ctx, input)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("Error checking for storage resource locality: %s", err.Error())
			return false, err
		}
		if hasStorage {
			foundInputs++
		}
	}

	if foundInputs >= len(job.Inputs) {
		log.Ctx(ctx).Trace().Msgf("Found %d of %d inputs - accepting job", foundInputs, len(job.Inputs))
		return true, nil
	} else {
		log.Ctx(ctx).Trace().Msgf("Found %d of %d inputs - passing on job", foundInputs, len(job.Inputs))
		return false, nil
	}
}

// the compute node "SelectJob" function will call out to this to handle
// applying the policy to the incoming job
// we are also given the executor so we can enquire about data locality
func ApplyJobSelectionPolicy(
	ctx context.Context,
	policy JobSelectionPolicy,
	e executor.Executor,
	data JobSelectionPolicyProbeData,
) (bool, error) {
	if policy.ProbeExec != "" {
		return applyJobSelectionPolicyExecProbe(ctx, policy.ProbeExec, data)
	} else if policy.ProbeHTTP != "" {
		return applyJobSelectionPolicyHTTPProbe(ctx, policy.ProbeHTTP, data)
	} else {
		return applyJobSelectionPolicySettings(ctx, policy, e, data.Spec)
	}
}
