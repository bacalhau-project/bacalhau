//go:build unit || !integration

package concurrency

import (
	"context"
	"errors"
	"testing"
)

// mockTransform doubles the input value if no error occurs.
func mockTransform(in int) (int, error) {
	return in * 2, nil
}

// TestAsyncChannelTransform_Basic checks if the transform function is correctly applied to valid input messages.
func TestAsyncChannelTransform_Basic(t *testing.T) {
	ctx := context.Background()
	input := make(chan *AsyncResult[int])
	go func() {
		input <- &AsyncResult[int]{Value: 1}
		input <- &AsyncResult[int]{Value: 2}
		close(input)
	}()

	output := AsyncChannelTransform(ctx, input, 10, mockTransform)

	var results []*AsyncResult[int]
	for res := range output {
		results = append(results, res)
	}

	if len(results) != 2 || results[0].Value != 2 || results[1].Value != 4 || results[0].Err != nil || results[1].Err != nil {
		t.Errorf("Unexpected transformation results: %+v", results)
	}
}

// TestAsyncChannelTransform_ErrorPropagation checks if errors in input messages are correctly propagated.
func TestAsyncChannelTransform_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	inputError := errors.New("input error")
	input := make(chan *AsyncResult[int])
	go func() {
		input <- &AsyncResult[int]{Err: inputError}
		close(input)
	}()

	output := AsyncChannelTransform(ctx, input, 10, mockTransform)

	result := <-output
	if result.Err == nil || result.Err != inputError {
		t.Errorf("Expected error to be propagated, got: %+v", result.Err)
	}
}

// TestAsyncChannelTransform_ContextCancellation checks if the operation stops gracefully upon context cancellation.
func TestAsyncChannelTransform_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := make(chan *AsyncResult[int])
	cancel()

	output := AsyncChannelTransform(ctx, input, 10, mockTransform)

	_, ok := <-output
	if ok {
		t.Errorf("Expected output channel to be closed due to context cancellation")
	}
}
