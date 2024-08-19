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

// NewAsyncValue creates a new AsyncResult with a value
func NewAsyncValue[T any](value T) *AsyncResult[T] {
	return &AsyncResult[T]{
		Value: value,
	}
}

// NewAsyncError creates a new AsyncResult with an error
func NewAsyncError[T any](err error) *AsyncResult[T] {
	return &AsyncResult[T]{
		Err: err,
	}
}

// NewAsyncResult creates a new AsyncResult with a value and an error
func NewAsyncResult[T any](value T, err error) *AsyncResult[T] {
	return &AsyncResult[T]{
		Value: value,
		Err:   err,
	}
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
		ar.Err = fmt.Errorf("%s", aux.Error)
	}
	return nil
}

func (ar *AsyncResult[T]) ValueOrError() (T, error) {
	return ar.Value, ar.Err
}

func errorToString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
