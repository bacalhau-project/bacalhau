package bacerrors

import (
	"fmt"
)

type JobNotFound BacalhauError

func NewJobNotFound(id string) *JobNotFound {
	var e JobNotFound
	e.SetID(id)
	return &e
}

func (e *JobNotFound) Msg(s string) string {
	return fmt.Sprintf("Job not found. ID: %s", s)
}
func (e *JobNotFound) SetMessage(s string) {
	e.theMessage = s
}

func (e *JobNotFound) Error() string {
	return e.serumError.Error()
}

func (e *JobNotFound) SetError(err error) {
	e.theCause = err
}

func (e *JobNotFound) Code() string {
	return ErrorCodeJobNotFound
}

func (e *JobNotFound) SetID(s string) {
	e.details["id"] = s
}
func (e *JobNotFound) GetID() string {
	if id, ok := e.details["id"]; ok {
		return id.(string)
	}
	return ""
}

const (
	ErrorCodeJobNotFound = "error-job-not-found"
)
