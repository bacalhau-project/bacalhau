package bacerrors

type GenericError struct {
	Code    string                 `json:"code"` //nolint:unused
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
	Err     error                  `json:"error"`
}

func NewGenericError(err error) *GenericError {
	var e GenericError
	e.Code = ErrorCodeGenericError
	e.Details = make(map[string]interface{})
	e.Err = err
	return &e
}

func (e *GenericError) GetMessage() string {
	return e.Message
}
func (e *GenericError) SetMessage(s string) {
	e.Message = s
}

func (e *GenericError) Error() string {
	return e.GetError().Error()
}
func (e *GenericError) GetError() error {
	return e.Err
}
func (e *GenericError) SetError(err error) {
	e.Err = err
}

func (e *GenericError) GetCode() string {
	return ErrorCodeGenericError
}
func (e *GenericError) SetCode(string) {
	e.Code = ErrorCodeGenericError
}

func (e *GenericError) GetDetails() map[string]interface{} {
	return e.Details
}

const (
	ErrorCodeGenericError = "error-generic-error"
)

var _ BacalhauErrorInterface = (*GenericError)(nil)
