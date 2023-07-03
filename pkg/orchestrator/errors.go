package orchestrator

// ErrSchedulerNotFound is returned when the scheduler is not found for a given evaluation type
type ErrSchedulerNotFound struct {
	EvaluationType string
}

func NewErrSchedulerNotFound(evaluationType string) ErrSchedulerNotFound {
	return ErrSchedulerNotFound{EvaluationType: evaluationType}
}

func (e ErrSchedulerNotFound) Error() string {
	return "scheduler not found for evaluation type: " + e.EvaluationType
}
