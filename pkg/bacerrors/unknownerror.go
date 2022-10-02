package bacerrors

type UnknownServerError GenericError

func NewUnknownServerError(msg string) *UnknownServerError {
	var e UnknownServerError
	e.Details = make(map[string]interface{})
	e.SetMessage(msg)
	return &e
}

func (e *UnknownServerError) GetMessage() string {
	return e.Message
}
func (e *UnknownServerError) SetMessage(s string) {
	e.Message = s
}

func (e *UnknownServerError) Error() string {
	return e.GetError().Error()
}
func (e *UnknownServerError) GetError() error {
	return e.Err
}
func (e *UnknownServerError) SetError(err error) {
	e.Err = err
}

func (e *UnknownServerError) GetCode() string {
	return ErrorCodeUnknownServerError
}
func (e *UnknownServerError) SetCode(string) {
	e.Code = ErrorCodeUnknownServerError
}

func (e *UnknownServerError) GetDetails() map[string]interface{} {
	return e.Details
}

const (
	ErrorCodeUnknownServerError = "error-unknown-server-error"
)

var _ BacalhauErrorInterface = (*UnknownServerError)(nil)
