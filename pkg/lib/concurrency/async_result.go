package concurrency

import (
	"encoding/json"
	"fmt"
)

// AsyncResult holds a value and an error, useful for async operations
type AsyncResult[T any] struct {
	Value T     `json:"value"`
	Err   error `json:"error,omitempty"`
}

// MarshalJSON customizes the JSON representation of AsyncResult
func (ar AsyncResult[T]) MarshalJSON() ([]byte, error) {
	type Alias AsyncResult[T]
	return json.Marshal(&struct {
		Error string `json:"error,omitempty"`
		*Alias
	}{
		Error: errorToString(ar.Err),
		Alias: (*Alias)(&ar),
	})
}

// UnmarshalJSON customizes the JSON unmarshalling of AsyncResult
func (ar *AsyncResult[T]) UnmarshalJSON(data []byte) error {
	type Alias AsyncResult[T]
	aux := &struct {
		Error string `json:"error,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(ar),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Error != "" {
		ar.Err = fmt.Errorf(aux.Error)
	}
	return nil
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
