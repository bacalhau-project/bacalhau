package types

// a representation of some data on a storage engine
// this opens up jobs that could operate on different types
// of storage at once
type StorageSpec struct {
	// e.g. ipfs, filecoin or s3
	// this can be empty for output volumes
	Engine string `json:"engine"`
	// we name input and output volumes so we can reference them
	// (possibly in dags and pipelines)
	// output volumes must have names
	Name string `json:"name"`
	// the id of the storage resource (e.g. cid in the case of ipfs)
	// this is empty in the case of outputs
	Cid string `json:"cid"`
	// for compute engines that "mount" the storage as filesystems (e.g. docker)
	// what path should we mount the storage to
	// this can be "stdout", "stderr" or "stdin"
	Path string `json:"path"`
}

// a storage entity that is consumed are produced by a job
// input storage specs are turned into storage volumes by drivers
// for example - the input storage spec might be ipfs cid XXX
// and a driver will turn that into a host path that can be consumed by a job
// another example - a wasm storage driver references the upstream ipfs
// cid (source) that can be streamed via a library call using the target name
// put simply - the nature of a storage volume depends on it's use by the
// executor engine
type StorageVolume struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// for VM style executors
type JobSpecVm struct {
	// this should be pullable by docker
	Image string `json:"image"`
	// optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint"`
	// a map of env to run the container with
	Env []string `json:"env"`
	// https://github.com/BTBurke/k8sresource strings
	Cpu    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}

// for Wasm style executors
type JobSpecWasm struct {
	Bytecode StorageSpec `json:"bytecode"`
}

// what we pass off to the executor to "run" the job
type JobSpec struct {
	// e.g. firecracker, docker or wasm
	Engine string `json:"engine"`

	// e.g. ipfs or localfs
	// these verifiers both just copy the results
	// and don't do any verification
	Verifier string `json:"verifier"`

	// for VM based executors
	Vm   JobSpecVm   `json:"job_spec_vm"`
	Wasm JobSpecWasm `json:"job_spec_wasm"`

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []StorageSpec `json:"inputs"`
	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec `json:"outputs"`
}

// keep track of job states on a particular node
type JobState struct {
	State     JobStateType `json:"state"`
	Status    string       `json:"status"`
	ResultsId string       `json:"results_id"`
}

// omly the client can update this as it's the client that will
// pay out based on the deal
type JobDeal struct {
	// how many nodes do we want to run this job?
	Concurrency int `json:"concurrency"`
	// the nodes we have assigned (and will pay)
	// other nodes are welcome to submit results without having been assigned
	// this is how they can bootstrap their reputation
	AssignedNodes []string `json:"assigned_nodes"`
}

// the view of a single job
// multiple compute nodes will be running this job
type Job struct {
	Id string `json:"id"`
	// the client node that "owns" this job (as in who submitted it)
	Owner string   `json:"owner"`
	Spec  *JobSpec `json:"spec"`
	Deal  *JobDeal `json:"deal"`
	// a map of nodeId -> state of the job on that node
	State map[string]*JobState `json:"state"`
}

// we emit these to other nodes so they update their
// state locally and can emit events locally
type JobEvent struct {
	JobId     string       `json:"job_id"`
	NodeId    string       `json:"node_id"`
	EventName JobEventType `json:"event_name"`
	// this is only defined in "create" events
	JobSpec *JobSpec `json:"job_spec"`
	// this is only defined in "update_deal" events
	JobDeal *JobDeal `json:"job_deal"`
	// most other events are a case of a client<->node state change
	JobState *JobState `json:"job_state"`
}

type ResultsList struct {
	Node   string `json:"node"`
	Cid    string `json:"cid"`
	Folder string `json:"folder"`
}
