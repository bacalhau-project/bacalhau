package types

type JobSpec struct {
	Id       string
	Cids     []string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}

type JobState struct {
	JobId     string
	NodeId    string
	State     string
	Status    string
	ResultCid string
}

// the view of a single job
// multiple compute nodes will be running this job
type JobData struct {
	Job   *JobSpec
	State map[string]*JobState
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs map[string]*JobData
}
