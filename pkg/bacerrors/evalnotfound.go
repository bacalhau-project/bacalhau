package bacerrors

import (
	"fmt"
)

type EvaluationNotFound GenericError

func NewEvaluationNotFound(id string) *EvaluationNotFound {
	var e EvaluationNotFound
	e.Code = ErrorCodeEvaluationNotFound
	e.Message = fmt.Sprintf(ErrorMessageEvaluationNotFound, id)
	e.Details = make(map[string]interface{})
	e.Details["id"] = id
	e.SetID(id)
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *EvaluationNotFound) GetMessage() string {
	return e.Message
}
func (e *EvaluationNotFound) SetMessage(s string) {
	e.Message = s
}

func (e *EvaluationNotFound) Error() string {
	return e.GetError().Error()
}
func (e *EvaluationNotFound) GetError() error {
	return e.Err
}
func (e *EvaluationNotFound) SetError(err error) {
	e.Err = err
}

func (e *EvaluationNotFound) GetCode() string {
	return ErrorCodeJobNotFound
}
func (e *EvaluationNotFound) SetCode(string) {
	e.Code = ErrorCodeJobNotFound
}

func (e *EvaluationNotFound) GetDetails() map[string]interface{} {
	return e.Details
}

func (e *EvaluationNotFound) GetID() string {
	if id, ok := e.Details["id"]; ok {
		return id.(string)
	}
	return ""
}
func (e *EvaluationNotFound) SetID(s string) {
	e.Details["id"] = s
}

const (
	ErrorCodeEvaluationNotFound = "error-evaluation-not-found"

	ErrorMessageEvaluationNotFound = "Evaluation not found. ID: %s"
)

var _ BacalhauErrorInterface = (*EvaluationNotFound)(nil)
