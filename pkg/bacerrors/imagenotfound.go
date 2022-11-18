package bacerrors

import (
	"fmt"
)

type ImageNotFound GenericError

func NewImageNotFound(id string) *ImageNotFound {
	var e ImageNotFound
	e.Code = ErrorCodeImageNotFound
	e.Message = fmt.Sprintf(ErrorMessageImageNotFound, id)
	e.Details = make(map[string]interface{})
	e.SetError(fmt.Errorf("%s", e.Message))
	return &e
}

func (e *ImageNotFound) GetMessage() string {
	return e.Message
}
func (e *ImageNotFound) SetMessage(s string) {
	e.Message = s
}

func (e *ImageNotFound) Error() string {
	return e.GetError().Error()
}
func (e *ImageNotFound) GetError() error {
	return e.Err
}
func (e *ImageNotFound) SetError(err error) {
	e.Err = err
}

func (e *ImageNotFound) GetCode() string {
	return ErrorCodeImageNotFound
}
func (e *ImageNotFound) SetCode(string) {
	e.Code = ErrorCodeImageNotFound
}

func (e *ImageNotFound) GetDetails() map[string]interface{} {
	return e.Details
}

func (e *ImageNotFound) GetImageName() string {
	if id, ok := e.Details["imagename"]; ok {
		return id.(string)
	}
	return ""
}
func (e *ImageNotFound) SetImageName(s string) {
	e.Details["imagename"] = s
}

const (
	ErrorCodeImageNotFound = "error-image-not-found"

	ErrorMessageImageNotFound = "Image not found. Image: %s"
)

var _ BacalhauErrorInterface = (*ImageNotFound)(nil)
