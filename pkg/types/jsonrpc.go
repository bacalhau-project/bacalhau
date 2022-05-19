package types

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
