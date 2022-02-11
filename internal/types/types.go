package types

type Job struct {
	Id       string
	Cids     []string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}

// a message from a peer on the network updating about a job on a node
type Update struct {
	JobId  string
	NodeId string
	State  string
	Status string
	Output string
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs       []Job
	JobState   map[string]map[string]string
	JobStatus  map[string]map[string]string
	JobResults map[string]map[string]string
}
