package node

const (
	// computeToOrchestratorDispatcherWatcherID is the ID of the watcher that listens for execution events
	// and forwards them to the orchestrator.
	computeToOrchestratorDispatcherWatcherID = "orchestrator-dispatcher"

	// computeExecutionHandlerWatcherID is the ID of the watcher that listens for execution events
	// and handles them locally by triggering the executor or bidder for example.
	computeExecutionHandlerWatcherID = "execution-handler"

	// computeExecutionLoggerWatcherID is the ID of the watcher that listens for execution events
	// and logs them.
	computeExecutionLoggerWatcherID = "compute-logger"

	// orchestratorToComputeDispatcherWatcherID is the ID of the watcher that listens for execution events
	// and forwards them to the compute nodes.
	orchestratorToComputeDispatcherWatcherID = "compute-dispatcher"

	// orchestratorEvaluationWatcherID is the ID of the watcher that listens for evaluation events
	// and enqueues them into the evaluation broker.
	orchestratorEvaluationWatcherID = "evaluation-watcher"

	// orchestratorExecutionLoggerWatcherID is the ID of the watcher that listens for execution events
	// and logs them.
	orchestratorExecutionLoggerWatcherID = "orchestrator-logger"
)
