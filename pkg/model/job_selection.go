package model

// Job selection policy configuration
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
	RejectStatelessJobs bool `json:"rejectStatelessJobs"`
	// should we accept jobs that specify networking
	// the default is "reject"
	AcceptNetworkedJobs bool `json:"acceptNetworkedJobs"`
	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHTTP string `json:"probeHTTP,omitempty"`
	ProbeExec string `json:"probeExec,omitempty"`
}

// generate a default empty job selection policy
func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{}
}
