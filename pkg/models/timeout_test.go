//go:build unit || !integration

package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TimeoutConfigTestSuite struct {
	suite.Suite
}

func (suite *TimeoutConfigTestSuite) TestGetters() {
	config := &TimeoutConfig{
		ExecutionTimeout: 10,
		QueueTimeout:     20,
		TotalTimeout:     30,
	}
	suite.Equal(10*time.Second, config.GetExecutionTimeout(), "Execution timeout should be 10 seconds")
	suite.Equal(20*time.Second, config.GetQueueTimeout(), "Queue timeout should be 20 seconds")
	suite.Equal(30*time.Second, config.GetTotalTimeout(), "Total timeout should be 30 seconds")
}

func (suite *TimeoutConfigTestSuite) TestCopy() {
	original := &TimeoutConfig{
		ExecutionTimeout: 10,
		QueueTimeout:     20,
		TotalTimeout:     30,
	}
	copyConfig := original.Copy()
	suite.Equal(original, copyConfig, "Copied config should be equal to the original")
	suite.NotSame(original, copyConfig, "Copied config should not be the same instance as the original")
}
func (suite *TimeoutConfigTestSuite) TestValidate() {
	tests := []struct {
		name      string
		config    *TimeoutConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "ValidConfig",
			config: &TimeoutConfig{
				ExecutionTimeout: 10,
				QueueTimeout:     20,
				TotalTimeout:     30,
			},
			expectErr: false,
		},
		{
			name:      "AllZeros",
			config:    &TimeoutConfig{},
			expectErr: false,
		},
		{
			name: "NoTotalTimeout",
			config: &TimeoutConfig{
				ExecutionTimeout: 10,
				QueueTimeout:     20,
			},
			expectErr: false,
		},
		{
			name: "OnlyTotalTimeout",
			config: &TimeoutConfig{
				TotalTimeout: 30,
			},
			expectErr: false,
		},
		{
			name:      "NilConfig",
			config:    nil,
			expectErr: true,
		},
		{
			name: "NegativeExecutionTimeout",
			config: &TimeoutConfig{
				ExecutionTimeout: -10,
				QueueTimeout:     20,
				TotalTimeout:     30,
			},
			expectErr: true,
			errMsg:    "invalid execution timeout value",
		},
		{
			name: "NegativeQueueTimeout",
			config: &TimeoutConfig{
				ExecutionTimeout: 10,
				QueueTimeout:     -20,
				TotalTimeout:     30,
			},
			expectErr: true,
			errMsg:    "invalid queue timeout value",
		},
		{
			name: "NegativeTotalTimeout",
			config: &TimeoutConfig{
				ExecutionTimeout: 10,
				QueueTimeout:     20,
				TotalTimeout:     -30,
			},
			expectErr: true,
			errMsg:    "invalid total timeout value",
		},
		{
			name: "InvalidTotalTimeout",
			config: &TimeoutConfig{
				ExecutionTimeout: 15,
				QueueTimeout:     20,
				TotalTimeout:     30,
			},
			expectErr: true,
			errMsg:    "execution timeout 15s and queue timeout 20s should be less than total timeout 30s",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.config.Validate()
			if tt.expectErr {
				suite.Error(err)
				suite.Contains(err.Error(), tt.errMsg)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func TestTimeoutConfigTestSuite(t *testing.T) {
	suite.Run(t, new(TimeoutConfigTestSuite))
}
