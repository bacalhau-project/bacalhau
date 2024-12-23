//go:build unit || !integration

package dispatcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (suite *ConfigTestSuite) TestDefaultConfig() {
	config := DefaultConfig()
	suite.Equal(defaultCheckpointInterval, config.CheckpointInterval)
	suite.Equal(defaultCheckpointTimeout, config.CheckpointTimeout)
	suite.Equal(defaultStallTimeout, config.StallTimeout)
	suite.Equal(defaultStallCheckInterval, config.StallCheckInterval)
	suite.Equal(defaultProcessInterval, config.ProcessInterval)
	suite.Equal(defaultSeekTimeout, config.SeekTimeout)
	suite.Equal(defaultBaseRetryInterval, config.BaseRetryInterval)
	suite.Equal(defaultMaxRetryInterval, config.MaxRetryInterval)
}

func (suite *ConfigTestSuite) TestConfigValidation() {
	valid := Config{
		CheckpointInterval: time.Second,
		CheckpointTimeout:  time.Second,
		StallTimeout:       time.Minute,
		StallCheckInterval: time.Second,
		ProcessInterval:    time.Millisecond,
		SeekTimeout:        time.Second,
		BaseRetryInterval:  time.Second,
		MaxRetryInterval:   time.Minute,
	}

	testCases := []struct {
		name        string
		mutate      func(*Config)
		expectError string
	}{
		{
			name:        "valid config",
			mutate:      func(*Config) {},
			expectError: "",
		},
		{
			name:        "zero values",
			mutate:      func(c *Config) { *c = Config{} },
			expectError: "must be positive",
		},
		{
			name:        "zero stall timeout",
			mutate:      func(c *Config) { c.StallTimeout = 0 },
			expectError: "StallTimeout must be positive",
		},
		{
			name: "invalid retry intervals",
			mutate: func(c *Config) {
				c.BaseRetryInterval = time.Minute
				c.MaxRetryInterval = time.Second
			},
			expectError: "MaxRetryInterval must be greater than or equal to BaseRetryInterval",
		},
		{
			name: "invalid stall intervals",
			mutate: func(c *Config) {
				c.StallTimeout = time.Second
				c.StallCheckInterval = time.Minute
			},
			expectError: "StallCheckInterval must be less than StallTimeout",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cfg := valid
			tc.mutate(&cfg)
			err := cfg.Validate()
			if tc.expectError == "" {
				suite.NoError(err)
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expectError)
			}
		})
	}
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
