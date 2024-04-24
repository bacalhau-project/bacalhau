//go:build unit || !integration

package models

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type BaseErrorTestSuite struct {
	suite.Suite
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithMessage() {
	message := "TestMessage"
	err := NewBaseError(message)

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithFormattedMessage() {
	// test that NewBaseError can accept a message with format specifiers
	message := "TestMessage %s"
	err := NewBaseError(message, "withFormat")
	suite.Equal("TestMessage withFormat", err.Error())
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithHint() {
	message := "TestMessage"
	hint := "TestHint"
	err := NewBaseError(message).WithHint(hint)

	suite.Equal(message, err.Error())
	suite.Equal(hint, err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithRetryable() {
	message := "TestMessage"
	err := NewBaseError(message).WithRetryable()

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.True(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithFailsExecution() {
	message := "TestMessage"
	err := NewBaseError(message).WithFailsExecution()

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.True(err.FailsExecution())
	suite.Nil(err.Details())
}

func (suite *BaseErrorTestSuite) TestBaseErrorWithDetails() {
	message := "TestMessage"
	details := map[string]string{"key1": "value1", "key2": "value2"}
	err := NewBaseError(message).WithDetails(details)

	suite.Equal(message, err.Error())
	suite.Empty(err.Hint())
	suite.False(err.Retryable())
	suite.False(err.FailsExecution())
	suite.Equal(details, err.Details())
}

func TestBaseErrorTestSuite(t *testing.T) {
	suite.Run(t, new(BaseErrorTestSuite))
}
