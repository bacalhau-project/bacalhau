package bacerrors

import (
	"fmt"
)

type DuplicateJob GenericError

func NewDuplicateJob(id string) *DuplicateJob {
	var e DuplicateJob
	e.Code = ErrorCodeDuplicateJob
	e.Message = fmt.Sprintf(ErrorMessageDuplicateJob, id)
	e.Details = make(map[string]interface{})
	e.Details["id"] = id
	e.SetID(id)
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *DuplicateJob) GetMessage() string {
	return e.Message
}
func (e *DuplicateJob) SetMessage(s string) {
	e.Message = s
}

func (e *DuplicateJob) Error() string {
	return e.GetError().Error()
}
func (e *DuplicateJob) GetError() error {
	return e.Err
}
func (e *DuplicateJob) SetError(err error) {
	e.Err = err
}

func (e *DuplicateJob) GetCode() string {
	return ErrorCodeDuplicateJob
}
func (e *DuplicateJob) SetCode(string) {
	e.Code = ErrorCodeDuplicateJob
}

func (e *DuplicateJob) GetDetails() map[string]interface{} {
	return e.Details
}

func (e *DuplicateJob) GetID() string {
	if id, ok := e.Details["id"]; ok {
		return id.(string)
	}
	return ""
}
func (e *DuplicateJob) SetID(s string) {
	e.Details["id"] = s
}

const (
	ErrorCodeDuplicateJob = "duplicate-job-found"

	ErrorMessageDuplicateJob = "Duplicate jobs found for ID: %s"
)

var _ BacalhauErrorInterface = (*DuplicateJob)(nil)
