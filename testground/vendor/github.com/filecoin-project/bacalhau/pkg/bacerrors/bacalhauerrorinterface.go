package bacerrors

type BacalhauErrorInterface interface {
	Error() string

	GetError() error
	SetError(error)

	GetCode() string
	SetCode(string)

	GetMessage() string
	SetMessage(string)

	GetDetails() map[string]interface{}
}
