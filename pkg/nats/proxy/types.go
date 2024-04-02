package proxy

type BaseRequest[T any] struct {
	TargetNodeID string
	Method       string
	Body         T
}

// ComputeEndpoint return the compute endpoint for the base request.
func (r *BaseRequest[T]) ComputeEndpoint() string {
	return computeEndpointPublishSubject(r.TargetNodeID, r.Method)
}

// OrchestratorEndpoint return the orchestrator endpoint for the base request.
func (r *BaseRequest[T]) OrchestratorEndpoint() string {
	return callbackPublishSubject(r.TargetNodeID, r.Method)
}
