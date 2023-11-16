package bacerrors

import (
	"fmt"
)

type MultipleJobsFound GenericError

func NewMultipleJobsFound(id string, matchingJobIDs []string) *MultipleJobsFound {
	var e MultipleJobsFound
	if len(matchingJobIDs) > 3 {
		matchingJobIDs = append(matchingJobIDs[:3], "...")
	}
	e.Code = ErrorCodeMultipleJobsFound
	e.Message = fmt.Sprintf(ErrorMessageMultipleJobsFound, id, matchingJobIDs)
	e.Details = make(map[string]interface{})
	e.Details["id"] = id
	e.Details["matchingJobIDs"] = matchingJobIDs
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *MultipleJobsFound) GetMessage() string {
	return e.Message
}
func (e *MultipleJobsFound) SetMessage(s string) {
	e.Message = s
}

func (e *MultipleJobsFound) Error() string {
	return e.GetError().Error()
}
func (e *MultipleJobsFound) GetError() error {
	return e.Err
}
func (e *MultipleJobsFound) SetError(err error) {
	e.Err = err
}

func (e *MultipleJobsFound) GetCode() string {
	return ErrorCodeMultipleJobsFound
}
func (e *MultipleJobsFound) SetCode(string) {
	e.Code = ErrorCodeMultipleJobsFound
}

func (e *MultipleJobsFound) GetDetails() map[string]interface{} {
	return e.Details
}

const (
	ErrorCodeMultipleJobsFound = "ambiguous-job-id"

	ErrorMessageMultipleJobsFound = "Multiple jobs found for jobID prefix: %s, matching jobIDs: %v"
)

var _ BacalhauErrorInterface = (*MultipleJobsFound)(nil)
