package bacerrors

import (
	"fmt"
)

type JobNotFound GenericError

func NewJobNotFound(id string) *JobNotFound {
	var e JobNotFound
	e.Code = ErrorCodeJobNotFound
	e.Message = fmt.Sprintf(ErrorMessageJobNotFound, id)
	e.Details = make(map[string]interface{})
	e.Details["id"] = id
	e.SetID(id)
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *JobNotFound) GetMessage() string {
	return e.Message
}
func (e *JobNotFound) SetMessage(s string) {
	e.Message = s
}

func (e *JobNotFound) Error() string {
	return e.GetError().Error()
}
func (e *JobNotFound) GetError() error {
	return e.Err
}
func (e *JobNotFound) SetError(err error) {
	e.Err = err
}

func (e *JobNotFound) GetCode() string {
	return ErrorCodeJobNotFound
}
func (e *JobNotFound) SetCode(string) {
	e.Code = ErrorCodeJobNotFound
}

func (e *JobNotFound) GetDetails() map[string]interface{} {
	return e.Details
}

func (e *JobNotFound) GetID() string {
	if id, ok := e.Details["id"]; ok {
		return id.(string)
	}
	return ""
}
func (e *JobNotFound) SetID(s string) {
	e.Details["id"] = s
}

const (
	ErrorCodeJobNotFound = "error-job-not-found"

	ErrorMessageJobNotFound = "Job not found. ID: %s"
)

var _ BacalhauErrorInterface = (*JobNotFound)(nil)
