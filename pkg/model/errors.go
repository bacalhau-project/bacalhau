package model

type JobNotFound struct {
	ID string
	//nolint:unused
	err error
}

func (e *JobNotFound) Error() string { return "Job not found. ID: " + e.ID }
