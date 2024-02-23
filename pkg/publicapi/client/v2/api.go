package client

// API is the root of the Bacalhau server API. The structure mirrors the
// structure of the API: each method returns an object that can submit requests
// to control a single part of the system.
type API interface {
	Agent() *Agent
	Auth() *Auth
	Jobs() *Jobs
	Nodes() *Nodes
}

type api struct {
	Client
}

func (c *api) Agent() *Agent {
	return &Agent{client: c.Client}
}

func (c *api) Auth() *Auth {
	return &Auth{client: c.Client}
}

func (c *api) Jobs() *Jobs {
	return &Jobs{client: c.Client}
}

func (c *api) Nodes() *Nodes {
	return &Nodes{client: c.Client}
}

func NewAPI(transport Client) API {
	return &api{Client: transport}
}

func New(address string, optFns ...OptionFn) API {
	return NewAPI(NewHTTPClient(address, optFns...))
}

var _ API = (*api)(nil)
