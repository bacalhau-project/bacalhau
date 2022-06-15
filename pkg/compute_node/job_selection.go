package compute_node

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/types"
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
	command string,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	return false, nil
}

func applyJobSelectionPolicyHttpProbe(
	url string,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	return false, nil
}

func applyJobSelectionPolicyDataSettings(
	policy JobSelectionDataPolicy,
	executor executor.Executor,
	job *types.JobSpec,
) (bool, error) {
	return false, nil
}

// the compute node "SelectJob" function will call out to this to handle
// applying the policy to the incoming job
// we are also given the executor so we can enquire about data locality
func ApplyJobSelectionPolicy(
	policy JobSelectionPolicy,
	executor executor.Executor,
	nodeId string,
	job *types.JobSpec,
) (bool, error) {
	if policy.ProbeExec != "" {
		return applyJobSelectionPolicyExecProbe(policy.ProbeExec, nodeId, job)
	} else if policy.ProbeHttp != "" {
		return applyJobSelectionPolicyHttpProbe(policy.ProbeHttp, nodeId, job)
	} else {
		return applyJobSelectionPolicyDataSettings(policy.Data, executor, job)
	}
}
