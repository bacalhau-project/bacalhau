package types

// a representation of some data on a storage engine
// this opens up jobs that could operate on different types
// of storage at once
type JobStorage struct {
	// e.g. ipfs, filecoin or s3
	Engine string
	Cid    string
}

// what we pass off to the executor to "run" the job
type JobSpec struct {
	// e.g. firecracker, docker or wasm
	Engine   string
	Commands []string
	Image    string
	Cpu      int
	Memory   int
	Disk     int
	// for example a list of IPFS cids (if we are using the IPFS storage engine)
	Inputs []JobStorage
}

// keep track of job states on a particular node
type JobState struct {
	State  string
	Status string
	// for example a list of IPFS cids (if we are using the IPFS storage engine)
	Outputs []JobStorage
}

// omly the client can update this as it's the client that will
// pay out based on the deal
type JobDeal struct {
	// how many nodes do we want to run this job?
	Concurrency int
	// the nodes we have assigned (and will pay)
	// other nodes are welcome to submit results without having been assigned
	// this is how they can bootstrap their reputation
	AssignedNodes []string
}

// the view of a single job
// multiple compute nodes will be running this job
type Job struct {
	Id string
	// the client node that "owns" this job (as in who submitted it)
	Owner string
	Spec  *JobSpec
	Deal  *JobDeal
	// a map of nodeId -> state of the job on that node
	State map[string]*JobState
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	JobId     string
	NodeId    string
	EventName string
	// this is only defined in "create" events
	JobSpec *JobSpec
	// this is only defined in "update_deal" events
	JobDeal *JobDeal
	// most other events are a case of a client<->node state change
	JobState *JobState
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs map[string]*Job
}
