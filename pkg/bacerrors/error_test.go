//go:build unit || !integration

package bacerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func (suite *ErrorTestSuite) TestNew() {
	err := New("test error")
	suite.Equal("test error", err.Error())
	suite.NotEmpty(err.StackTrace())
}

func (suite *ErrorTestSuite) TestErrorWithMessage() {
	message := "TestMessage"
	err := New(message)

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithFormattedMessage() {
	// test that New can accept a message with format specifiers
	message := "TestMessage %s"
	err := New(message, "withFormat")
	suite.Equal("TestMessage withFormat", err.Error())
}

func (suite *ErrorTestSuite) TestErrorWithHint() {
	message := "TestMessage"
	hint := "TestHint"
	err := New(message).WithHint(hint)

	suite.Equal(message, err.Error())
	suite.Equal(hint, err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithRetryable() {
	message := "TestMessage"
	err := New(message).WithRetryable()

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.True(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithFailsExecution() {
	message := "TestMessage"
	err := New(message).WithFailsExecution()

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.True(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithDetails() {
	message := "TestMessage"
	details := map[string]string{"key1": "value1", "key2": "value2"}
	err := New(message).WithDetails(details)

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Equal(details, err.Details())

	// Test appending details
	additionalDetails := map[string]string{"key3": "value3", "key2": "newvalue2"}
	err = err.WithDetails(additionalDetails)

	expectedDetails := map[string]string{"key1": "value1", "key2": "newvalue2", "key3": "value3"}
	suite.Equal(expectedDetails, err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithDetail() {
	message := "TestMessage"
	err := New(message).WithDetail("key1", "value1")

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Equal(map[string]string{"key1": "value1"}, err.Details())

	// Test adding another detail
	err = err.WithDetail("key2", "value2")
	expectedDetails := map[string]string{"key1": "value1", "key2": "value2"}
	suite.Equal(expectedDetails, err.Details())

	// Test overwriting an existing detail
	err = err.WithDetail("key1", "newvalue1")
	expectedDetails = map[string]string{"key1": "newvalue1", "key2": "value2"}
	suite.Equal(expectedDetails, err.Details())
}

func (suite *ErrorTestSuite) TestErrorWithCode() {
	message := "TestMessage"
	err := New(message).WithCode(BadRequestError)

	suite.Equal(message, err.Error())
	suite.Equal(BadRequestError, err.Code())
	suite.Equal(400, err.HTTPStatusCode()) // BadRequestError should map to 400
}

func (suite *ErrorTestSuite) TestErrorWithHTTPStatusCode() {
	message := "TestMessage"
	err := New(message).WithHTTPStatusCode(418) // I'm a teapot

	suite.Equal(message, err.Error())
	suite.Equal(418, err.HTTPStatusCode())
}

func (suite *ErrorTestSuite) TestErrorWithComponent() {
	message := "TestMessage"
	err := New(message).WithComponent("TestComponent")

	suite.Equal(message, err.Error())
	suite.Equal("TestComponent", err.Component())
}

func (suite *ErrorTestSuite) TestErrorChaining() {
	err := New("TestMessage").
		WithHint("TestHint").
		WithRetryable().
		WithFailsExecution().
		WithDetail("key1", "value1").
		WithCode(NotFoundError).
		WithHTTPStatusCode(404).
		WithComponent("TestComponent")

	suite.Equal("TestMessage", err.Error())
	suite.Equal("TestHint", err.Hint())
	suite.True(err.Retryable())
	suite.True(err.FailsExecution())
	suite.Equal(map[string]string{"key1": "value1"}, err.Details())
	suite.Equal(NotFoundError, err.Code())
	suite.Equal(404, err.HTTPStatusCode())
	suite.Equal("TestComponent", err.Component())
}

func (suite *ErrorTestSuite) TestWrapNonBacerror() {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, "wrapped error")

	suite.Equal("wrapped error: original error", wrappedErr.Error())
	suite.Equal("wrapped error: original error", wrappedErr.ErrorWrapped())
	suite.Equal(originalErr, errors.Unwrap(wrappedErr))
	suite.NotEmpty(wrappedErr.StackTrace())
}

func (suite *ErrorTestSuite) TestWrapBacerror() {
	originalErr := New("original error")
	wrappedErr := Wrap(originalErr, "wrapped error")

	suite.Equal("original error", wrappedErr.Error())
	suite.Equal("wrapped error: original error", wrappedErr.ErrorWrapped())
	suite.Equal(originalErr, errors.Unwrap(wrappedErr))
	suite.Equal(originalErr.StackTrace(), wrappedErr.StackTrace())
}

func (suite *ErrorTestSuite) TestMultipleWraps() {
	err1 := New("error1")
	err2 := Wrap(err1, "error2")
	err3 := Wrap(err2, "error3")

	suite.Equal("error1", err3.Error())
	suite.Equal("error3: error2: error1", err3.ErrorWrapped())

	unwrapped := errors.Unwrap(err3)
	suite.NotNil(unwrapped)
	if bacErr, ok := unwrapped.(Error); ok {
		suite.Equal("error1", bacErr.Error())
		suite.Equal("error2: error1", bacErr.ErrorWrapped())
	} else {
		suite.Fail("Unwrapped error is not of type *Error")
	}

	unwrapped = errors.Unwrap(unwrapped)
	if bacErr, ok := unwrapped.(Error); ok {
		suite.Equal("error1", bacErr.Error())
		suite.Equal("error1", bacErr.ErrorWrapped())
	} else {
		suite.Fail("Unwrapped error is not of type *Error")
	}

	unwrapped = errors.Unwrap(unwrapped)
	suite.Nil(unwrapped)
}

func (suite *ErrorTestSuite) TestWrapWithFormat() {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, "wrapped error: %s", "with format")

	suite.Equal("wrapped error: with format: original error", wrappedErr.Error())
	suite.Equal("wrapped error: with format: original error", wrappedErr.ErrorWrapped())
}

func TestErrorTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}
