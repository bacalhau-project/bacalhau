package models

import (
	"errors"
	"fmt"
	"time"
)

// TimeoutConfig is the configuration for timeout related settings,
// such as execution and shutdown timeouts.
type TimeoutConfig struct {
	// ExecutionTimeout is the maximum amount of time a task is allowed to run in seconds.
	// Zero means no timeout, such as for a daemon task.
	ExecutionTimeout int64 `json:"ExecutionTimeout,omitempty"`
	// QueueTimeout is the maximum amount of time a task is allowed to wait in the orchestrator
	// queue in seconds before being scheduled. Zero means no timeout.
	QueueTimeout int64 `json:"QueueTimeout,omitempty"`
	// TotalTimeout is the maximum amount of time a task is allowed to complete in seconds.
	// This includes the time spent in the queue, the time spent executing and the time spent retrying.
	// Zero means no timeout.
	TotalTimeout int64 `json:"TotalTimeout,omitempty"`
}

// GetExecutionTimeout returns the execution timeout duration
// Returns ExecutionTimeout if configured, otherwise returns TotalTimeout value, otherwise returns 0.
func (t *TimeoutConfig) GetExecutionTimeout() time.Duration {
	if t.ExecutionTimeout > 0 {
		return time.Duration(t.ExecutionTimeout) * time.Second
	}
	return time.Duration(t.TotalTimeout) * time.Second
}

// GetQueueTimeout returns the queue timeout duration
// Returns QueueTimeout if configured, otherwise returns 0.
// We don't fallback to TotalTimeout to allow users to disable queueing if no nodes were available.
func (t *TimeoutConfig) GetQueueTimeout() time.Duration {
	return time.Duration(t.QueueTimeout) * time.Second
}

// GetTotalTimeout returns the total timeout duration
func (t *TimeoutConfig) GetTotalTimeout() time.Duration {
	return time.Duration(t.TotalTimeout) * time.Second
}

// Copy returns a deep copy of the timeout config.
func (t *TimeoutConfig) Copy() *TimeoutConfig {
	if t == nil {
		return nil
	}
	return &TimeoutConfig{
		ExecutionTimeout: t.ExecutionTimeout,
		QueueTimeout:     t.QueueTimeout,
		TotalTimeout:     t.TotalTimeout,
	}
}

// Validate is used to check a timeout config for reasonable configuration.
// This is called after server side defaults are applied.
func (t *TimeoutConfig) Validate() error {
	mErr := t.ValidateSubmission()
	if t.TotalTimeout > 0 {
		if (t.ExecutionTimeout + t.QueueTimeout) > t.TotalTimeout {
			mErr = errors.Join(mErr, fmt.Errorf(
				"execution timeout %s and queue timeout %s should be less than total timeout %s",
				t.GetExecutionTimeout(), t.GetQueueTimeout(), t.GetTotalTimeout()))
		}
	}
	return mErr
}

// ValidateSubmission is used to check a timeout config for reasonable configuration when it is submitted.
func (t *TimeoutConfig) ValidateSubmission() error {
	if t == nil {
		return errors.New("missing timeout config")
	}
	var mErr error
	if t.ExecutionTimeout < 0 {
		mErr = errors.Join(mErr, fmt.Errorf("invalid execution timeout value: %s", t.GetExecutionTimeout()))
	}
	if t.QueueTimeout < 0 {
		mErr = errors.Join(mErr, fmt.Errorf("invalid queue timeout value: %s", t.GetQueueTimeout()))
	}
	if t.TotalTimeout < 0 {
		mErr = errors.Join(mErr, fmt.Errorf("invalid total timeout value: %s", t.GetTotalTimeout()))
	}
	return mErr
}
