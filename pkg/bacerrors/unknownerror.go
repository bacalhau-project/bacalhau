package bacerrors

type UnknownServerError BacalhauError

func NewUnknownServerError(msg string) *UnknownServerError {
	var e UnknownServerError
	e.SetMessage(msg)
	return &e
}

func (e *UnknownServerError) Message() string {
	return e.theMessage
}
func (e *UnknownServerError) SetMessage(s string) {
	e.theMessage = s
}

func (e *UnknownServerError) Error() string {
	return e.theCause.Error()
}
func (e *UnknownServerError) SetError(err error) {
	e.theCause = err
}

func (e *UnknownServerError) Code() string {
	return ErrorCodeUnknownServerError
}

func (e *UnknownServerError) Details() map[string]interface{} {
	return e.details
}

const (
	ErrorCodeUnknownServerError = "error-unknown-server-error"
)
