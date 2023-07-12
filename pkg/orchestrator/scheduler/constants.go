package scheduler

const (
	// execNotNeeded is the status used when a job no longer requires an execution
	execNotNeeded = "execution not needed due to job update"

	// execLost is the status used when an execution is lost
	execLost = "execution is lost since its node is down"
)
