package bacerrors

type ContextCanceledError GenericError

func NewContextCanceledError(msg string) *ContextCanceledError {
	var e ContextCanceledError
	e.Details = make(map[string]interface{})
	e.SetMessage(msg)
	return &e
}

func (e *ContextCanceledError) GetMessage() string {
	return e.Message
}
func (e *ContextCanceledError) SetMessage(s string) {
	e.Message = s
}

func (e *ContextCanceledError) Error() string {
	return e.GetError().Error()
}
func (e *ContextCanceledError) GetError() error {
	return e.Err
}
func (e *ContextCanceledError) SetError(err error) {
	e.Err = err
}

func (e *ContextCanceledError) GetCode() string {
	return ErrorCodeContextCanceledError
}
func (e *ContextCanceledError) SetCode(string) {
	e.Code = ErrorCodeContextCanceledError
}

func (e *ContextCanceledError) GetDetails() map[string]interface{} {
	return e.Details
}

const (
	ErrorCodeContextCanceledError = "error-context-canceled-error"
)

var _ BacalhauErrorInterface = (*ContextCanceledError)(nil)
