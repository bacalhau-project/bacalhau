package bacerrors

import (
	"encoding/json"
)

// JSONError is a struct used for JSON serialization of errorImpl
type JSONError struct {
	Cause          string            `json:"Cause"`
	Hint           string            `json:"Hint"`
	Retryable      bool              `json:"Retryable"`
	FailsExecution bool              `json:"FailsExecution"`
	Component      string            `json:"Component"`
	HTTPStatusCode int               `json:"HTTPStatusCode"`
	Details        map[string]string `json:"Details"`
	Code           ErrorCode         `json:"Code"`
}

// MarshalJSON implements the json.Marshaler interface
func (e *errorImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(&JSONError{
		Cause:          e.cause,
		Hint:           e.hint,
		Retryable:      e.retryable,
		FailsExecution: e.failsExecution,
		Component:      e.component,
		HTTPStatusCode: e.httpStatusCode,
		Details:        e.details,
		Code:           e.code,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *errorImpl) UnmarshalJSON(data []byte) error {
	var je JSONError
	if err := json.Unmarshal(data, &je); err != nil {
		return err
	}

	e.cause = je.Cause
	e.hint = je.Hint
	e.retryable = je.Retryable
	e.failsExecution = je.FailsExecution
	e.component = je.Component
	e.httpStatusCode = je.HTTPStatusCode
	e.details = je.Details
	e.code = je.Code

	return nil
}
