package types

// a representation of some data on a storage engine
// this opens up jobs that could operate on different types
// of storage at once
type StorageSpec struct {
	// e.g. ipfs, filecoin or s3
	Engine string
	// we name input and output volumes so we can reference them
	// (possibly in dags and pipelines)
	// output volumes must have names
	Name string
	// the id of the storage resource (e.g. cid in the case of ipfs)
	// this is empty in the case of outputs
	Cid string
	// for compute engines that "mount" the storage as filesystems (e.g. docker)
	// what path should we mount the storage to
	// this can be "stdout", "stderr" or "stdin"
	Path string
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
	Type   string
	Source string
	Target string
}

// for VM style executors
type JobSpecVm struct {
	// this should be pullable by docker
	Image string
	// optionally override the default entrypoint
	Entrypoint string
	// a map of env to run the container with
	Env []string
	// https://github.com/BTBurke/k8sresource strings
	Cpu    string
	Memory string
	Disk   string
}

// for Wasm style executors
type JobSpecWasm struct {
	Bytecode StorageSpec
}

// what we pass off to the executor to "run" the job
type JobSpec struct {
	// e.g. firecracker, docker or wasm
	Engine string

	// for VM based executors
	Vm   JobSpecVm
	Wasm JobSpecWasm

	// for WASM based executors
	Bytecode StorageSpec

	// the data volumes we will read in the job
	// for example "read this ipfs cid"
	Inputs []StorageSpec
	// the data volumes we will write in the job
	// for example "write the results to ipfs"
	Outputs []StorageSpec
}

// keep track of job states on a particular node
type JobState struct {
	State     string
	Status    string
	ResultsId string
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

type ResultsList struct {
	Node   string
	Cid    string
	Folder string
}

// JSON RPC

type ListArgs struct {
}

type SubmitArgs struct {
	Spec *JobSpec
	Deal *JobDeal
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs map[string]*Job
}
