package bacerrors

import (
	"fmt"
)

type AlreadyExists GenericError

func NewAlreadyExists(id string, typ string) *AlreadyExists {
	var e AlreadyExists
	e.Code = ErrorCodeAlreadyExists
	e.Message = fmt.Sprintf(ErrorMessageAlreadyExists, id, typ)
	e.Details = make(map[string]interface{})
	e.Details["id"] = id
	e.SetID(id)
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *AlreadyExists) GetMessage() string {
	return e.Message
}
func (e *AlreadyExists) SetMessage(s string) {
	e.Message = s
}

func (e *AlreadyExists) Error() string {
	return e.GetError().Error()
}
func (e *AlreadyExists) GetError() error {
	return e.Err
}
func (e *AlreadyExists) SetError(err error) {
	e.Err = err
}

func (e *AlreadyExists) GetCode() string {
	return ErrorCodeJobNotFound
}
func (e *AlreadyExists) SetCode(string) {
	e.Code = ErrorCodeJobNotFound
}

func (e *AlreadyExists) GetDetails() map[string]interface{} {
	return e.Details
}

func (e *AlreadyExists) GetID() string {
	if id, ok := e.Details["id"]; ok {
		return id.(string)
	}
	return ""
}
func (e *AlreadyExists) SetID(s string) {
	e.Details["id"] = s
}

const (
	ErrorCodeAlreadyExists    = "error-already-exists"
	ErrorMessageAlreadyExists = "%s (%s) already exists."
)

var _ BacalhauErrorInterface = (*AlreadyExists)(nil)
