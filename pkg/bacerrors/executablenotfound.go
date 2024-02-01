package bacerrors

import (
	"fmt"
)

type ExecutableNotFound GenericError

func NewExecutableNotFound(commandLine string) *ExecutableNotFound {
	var e ExecutableNotFound
	e.Code = ErrorCodeExecutableNotFound
	e.Message = fmt.Sprintf(ErrorMessageExecutableNotFound, commandLine)
	e.Details = make(map[string]interface{})
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *ExecutableNotFound) GetMessage() string {
	return e.Message
}
func (e *ExecutableNotFound) SetMessage(s string) {
	e.Message = s
}

func (e *ExecutableNotFound) Error() string {
	return e.GetError().Error()
}
func (e *ExecutableNotFound) GetError() error {
	return e.Err
}
func (e *ExecutableNotFound) SetError(err error) {
	e.Err = err
}

func (e *ExecutableNotFound) GetCode() string {
	return ErrorCodeExecutableNotFound
}
func (e *ExecutableNotFound) SetCode(string) {
	e.Code = ErrorCodeExecutableNotFound
}

func (e *ExecutableNotFound) GetDetails() map[string]interface{} {
	return e.Details
}

const (
	ErrorCodeExecutableNotFound = "error-executable-not-found"

	ErrorMessageExecutableNotFound = "Executable not found. Command: %s"
)

var _ BacalhauErrorInterface = (*ExecutableNotFound)(nil)
