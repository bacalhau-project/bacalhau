package types

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
	Probe Probe `json:"probe,omitempty"`
}
