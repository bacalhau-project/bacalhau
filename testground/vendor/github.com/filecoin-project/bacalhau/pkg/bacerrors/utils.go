package bacerrors

import (
	"encoding/json"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

const UnknownError = "error-unknown"

type ErrorResponse struct {
	Code    string                 `json:"Code"`
	Message string                 `json:"Message"`
	Details map[string]interface{} `json:"Details"`
	Err     string                 `json:"Err"`
}

func NewResponseUnknownError(err error) *ErrorResponse {
	e := &ErrorResponse{}
	e.Code = ErrorCodeUnknownServerError
	e.Message = err.Error()
	e.Details = map[string]interface{}{}
	e.Err = err.Error()
	return e
}

func (e *ErrorResponse) Error() string {
	return e.Message
}

func ErrorToErrorResponse(err error) string {
	e := ErrorToErrorResponseObject(err)
	return ConvertErrorToText(e)
}

func ErrorToErrorResponseObject(err error) *ErrorResponse {
	e := &ErrorResponse{}
	if err == nil {
		return e
	}

	if system.CheckIfObjectImplementsType(BacalhauErrorInterface(nil), err) {
		bacErr := err.(BacalhauErrorInterface)
		// Convert to ErrorResponse
		e = &ErrorResponse{
			Code:    bacErr.GetCode(),
			Message: bacErr.GetMessage(),
			Details: bacErr.GetDetails(),
			Err:     bacErr.GetError().Error(),
		}
	} else {
		// If not, then it's a generic error, so we need structure it as a ErrorResponse
		e.Code = ErrorCodeUnknownServerError
		e.Message = err.Error()
		e.Details = map[string]interface{}{}
		e.Err = err.Error()
	}

	return e
}

func ConvertErrorToText(err *ErrorResponse) string {
	str, marshalError := json.Marshal(err)
	if marshalError != nil {
		msg := "error converting BacalhauError to JSON"
		log.Error().Err(marshalError).Msg(msg)
		str = append(str, []byte("\n"+msg)...)
	}
	return string(str)
}
