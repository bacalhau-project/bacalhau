//go:build unit || !integration

package telemetry

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRecordErrorOnSpan(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpan(span)

	assert.False(t, span.ended)

	expectedErr := errors.New("dummy error")

	actualErr := f(expectedErr)

	assert.Equal(t, expectedErr, actualErr)

	assert.False(t, span.ended, "span should be closed by a defer statement in the calling function")
	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)
}

func TestRecordErrorOnSpanTwo(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanTwo[string](span)

	assert.False(t, span.ended)

	expectedParam := "blah"
	expectedErr := errors.New("dummy error")

	actualParam, actualErr := f(expectedParam, expectedErr)

	assert.Equal(t, expectedParam, actualParam)
	assert.Equal(t, expectedErr, actualErr)

	assert.False(t, span.ended, "span should be closed by a defer statement in the calling function")
	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)
}

func TestRecordErrorOnSpanThree(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanThree[string, string](span)

	assert.False(t, span.ended)

	expectedFirstParam := "foo"
	expectedSecondParam := "bar"
	expectedErr := errors.New("dummy error")

	actualFirstParam, actualSecondParam, actualErr := f(expectedFirstParam, expectedSecondParam, expectedErr)

	assert.Equal(t, expectedFirstParam, actualFirstParam)
	assert.Equal(t, expectedSecondParam, actualSecondParam)
	assert.Equal(t, expectedErr, actualErr)

	assert.False(t, span.ended, "span should be closed by a defer statement in the calling function")
	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)
}

func TestRecordErrorOnSpanReadCloserAndClose(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanReadCloserAndClose(span)

	assert.False(t, span.ended)

	expectedContent := "content"
	expectedErr := errors.New("dummy error")

	actualReader, actualErr := f(io.NopCloser(strings.NewReader("content")), expectedErr)

	assert.Equal(t, expectedErr, actualErr)

	actualContent, err := io.ReadAll(actualReader)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(actualContent))

	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)

	assert.False(t, span.ended, "span should be closed only once close has been called")
	assert.NoError(t, actualReader.Close())
	assert.True(t, span.ended, "span should be closed only once close has been called")
}

func TestRecordErrorOnSpanReadCloserTwoAndClose(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanReadCloserTwoAndClose[string](span)

	assert.False(t, span.ended)

	expectedContent := "content"
	expectedParam := "param"
	expectedErr := errors.New("dummy error")

	actualReader, actualFirstParam, actualErr := f(io.NopCloser(strings.NewReader("content")), expectedParam, expectedErr)

	assert.Equal(t, expectedErr, actualErr)
	assert.Equal(t, expectedParam, actualFirstParam)

	actualContent, err := io.ReadAll(actualReader)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(actualContent))

	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)

	assert.False(t, span.ended, "span should be closed only once close has been called")
	assert.NoError(t, actualReader.Close())
	assert.True(t, span.ended, "span should be closed only once close has been called")
}

func TestRecordErrorOnSpanOneChannel_withError(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanOneChannel[string](span)

	assert.False(t, span.ended)

	expectedErr := errors.New("dummy error")

	actualChan, actualErr := f(make(chan string), expectedErr)

	assert.Equal(t, expectedErr, actualErr)
	assert.Nil(t, actualChan)

	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)

	assert.True(t, span.ended, "span should be closed immediately as an error was returned")
}

func TestRecordErrorOnSpanOneChannel_withoutError(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanOneChannel[string](span)

	assert.False(t, span.ended)

	expectedParam := "param"

	realChan := make(chan string, 1)

	actualChan, actualErr := f(realChan, nil)

	assert.Nil(t, actualErr)

	assert.Empty(t, span.err)
	assert.Empty(t, span.statusCode)
	assert.Empty(t, span.statusDescription)

	assert.False(t, span.ended, "span should only be closed once the channel has received a response")
	realChan <- expectedParam
	actualParam := <-actualChan
	assert.True(t, span.ended, "span should only be closed once the channel has received a response")
	assert.Equal(t, expectedParam, actualParam)
}

func TestRecordErrorOnSpanTwoChannels_withError(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanTwoChannels[string](span)

	assert.False(t, span.ended)

	expectedErr := errors.New("dummy error")

	paramChan := make(chan string, 1)
	defer close(paramChan)
	errChan := make(chan error, 1)
	defer close(errChan)

	_, actualErrChan := f(paramChan, errChan)

	assert.False(t, span.ended, "span should only be closed once the channel has received a response")
	errChan <- expectedErr
	actualErr := <-actualErrChan
	assert.True(t, span.ended, "span should only be closed once the channel has received a response")

	assert.Equal(t, expectedErr, actualErr)
	assert.Equal(t, expectedErr, span.err)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, actualErr.Error(), span.statusDescription)
}

func TestRecordErrorOnSpanTwoChannels_withoutError(t *testing.T) {
	span := &testSpan{}
	f := RecordErrorOnSpanTwoChannels[string](span)

	assert.False(t, span.ended)

	expectedParam := "param"

	paramChan := make(chan string, 1)
	defer close(paramChan)
	errChan := make(chan error, 1)
	defer close(errChan)

	actualParamChan, _ := f(paramChan, errChan)

	assert.False(t, span.ended, "span should only be closed once the channel has received a response")
	paramChan <- expectedParam
	actualParam := <-actualParamChan
	assert.Eventually(t, func() bool {
		return span.ended
	}, 1*time.Second, 5*time.Millisecond, "span should only be closed once the channel has received a response")

	assert.Empty(t, span.err)
	assert.Empty(t, span.statusCode)
	assert.Empty(t, span.statusDescription)

	assert.Equal(t, expectedParam, actualParam)
}

var _ trace.Span = &testSpan{}

type testSpan struct {
	err               error
	ended             bool
	statusCode        codes.Code
	statusDescription string
}

func (t *testSpan) End(...trace.SpanEndOption) {
	t.ended = true
}

func (t *testSpan) AddEvent(string, ...trace.EventOption) {}

func (t *testSpan) IsRecording() bool {
	return true
}

func (t *testSpan) RecordError(err error, _ ...trace.EventOption) {
	t.err = err
}

func (t *testSpan) SpanContext() trace.SpanContext {
	return trace.SpanContext{}
}

func (t *testSpan) SetStatus(code codes.Code, description string) {
	t.statusCode = code
	t.statusDescription = description
}

func (t *testSpan) SetName(string) {}

func (t *testSpan) SetAttributes(...attribute.KeyValue) {}

func (t *testSpan) TracerProvider() trace.TracerProvider {
	return trace.NewNoopTracerProvider()
}
