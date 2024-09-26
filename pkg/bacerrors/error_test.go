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
