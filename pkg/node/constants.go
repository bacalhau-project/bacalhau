package node

const (
	// computeCallbackForwarderWatcherID is the ID of the watcher that listens for execution events
	// and forwards them to the orchestrator callback.
	computeCallbackForwarderWatcherID = "compute-callback-forwarder"

	// computeExecutionHandlerWatcherID is the ID of the watcher that listens for execution events
	// and handles them locally by triggering the executor or bidder for example.
	computeExecutionHandlerWatcherID = "compute-execution-handler"

	// computeExecutionLoggerWatcherID is the ID of the watcher that listens for execution events
	// and logs them.
	computeExecutionLoggerWatcherID = "compute-execution-logger"

	// orchestratorEvaluationWatcherID is the ID of the watcher that listens for evaluation events
	// and enqueues them into the evaluation broker.
	orchestratorEvaluationWatcherID = "evaluation-watcher"
)
