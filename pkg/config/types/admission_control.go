package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobAdmissionControl struct {
	// Locality specifies the locality of the job input data.
	Locality models.JobSelectionDataLocality `yaml:"Locality,omitempty" json:"Locality,omitempty"`
	// RejectStatelessJobs indicates whether to reject stateless jobs, i.e. jobs without inputs.
	RejectStatelessJobs bool `yaml:"RejectStatelessJobs,omitempty" json:"RejectStatelessJobs,omitempty"`
	// AcceptNetworkedJobs indicates whether to accept jobs that require network access.
	// Will be deprecated in v1.7 in favor of RejectNetworkedJobs.
	AcceptNetworkedJobs bool `yaml:"AcceptNetworkedJobs,omitempty" json:"AcceptNetworkedJobs,omitempty"`
	// RejectNetworkedJobs indicates whether to reject jobs that require network access.
	RejectNetworkedJobs bool `yaml:"RejectNetworkedJobs,omitempty" json:"RejectNetworkedJobs,omitempty"`
	// ProbeHTTP specifies the HTTP endpoint for probing job submission.
	ProbeHTTP string `yaml:"ProbeHTTP,omitempty" json:"ProbeHTTP,omitempty"`
	// ProbeExec specifies the command to execute for probing job submission.
	ProbeExec string `yaml:"ProbeExec,omitempty" json:"ProbeExec,omitempty"`
}
