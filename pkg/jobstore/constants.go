package jobstore

const (
	// EventObjectExecutionUpsert is the event type for execution upsert events, which holds richer data
	// about the execution's update, such as the previous and new execution data, and any events.
	EventObjectExecutionUpsert = "ExecutionUpsert"
	// EventObjectEvaluation is the event type for evaluation events
	EventObjectEvaluation = "Evaluation"
)
