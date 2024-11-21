//go:build unit || !integration

package ncl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type ConfigTestSuite struct {
	suite.Suite
	registry   *envelope.Registry
	serializer envelope.MessageSerializer
}

func (suite *ConfigTestSuite) SetupTest() {
	suite.registry = envelope.NewRegistry()
	suite.serializer = envelope.NewSerializer()
}

func (suite *ConfigTestSuite) TestPublisherConfigDefaults() {
	config := PublisherConfig{}
	config.setDefaults()

	suite.NotNil(config.MessageSerializer, "should set default serializer")
}

func (suite *ConfigTestSuite) TestPublisherConfigValidation() {
	testCases := []struct {
		name        string
		config      PublisherConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty config",
			config:      PublisherConfig{},
			expectError: true,
			errorMsg:    "name cannot be blank",
		},
		{
			name: "missing registry",
			config: PublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
			},
			expectError: true,
			errorMsg:    "message registry cannot be nil",
		},
		{
			name: "missing serializer",
			config: PublisherConfig{
				Name:            "test",
				MessageRegistry: suite.registry,
			},
			expectError: true,
			errorMsg:    "message serializer cannot be nil",
		},
		{
			name: "both destination and prefix",
			config: PublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				DestinationPrefix: "test.prefix",
			},
			expectError: true,
			errorMsg:    "cannot specify both destination and destination prefix",
		},
		{
			name: "valid with destination",
			config: PublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
			},
			expectError: false,
		},
		{
			name: "valid with prefix",
			config: PublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				DestinationPrefix: "test.prefix",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.config.Validate()
			if tc.expectError {
				suite.Error(err)
				suite.Contains(err.Error(), tc.errorMsg)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigTestSuite) TestOrderedPublisherConfigDefaults() {
	config := OrderedPublisherConfig{}
	config.setDefaults()

	defaults := DefaultOrderedPublisherConfig()
	suite.NotNil(config.MessageSerializer, "should set default serializer")
	suite.Equal(defaults.AckWait, config.AckWait)
	suite.Equal(defaults.MaxPending, config.MaxPending)
	suite.Equal(defaults.RetryAttempts, config.RetryAttempts)
	suite.Equal(defaults.RetryWait, config.RetryWait)
}

func (suite *ConfigTestSuite) TestOrderedPublisherConfigValidation() {
	testCases := []struct {
		name        string
		config      OrderedPublisherConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty config",
			config:      OrderedPublisherConfig{},
			expectError: true,
			errorMsg:    "name cannot be blank",
		},
		{
			name: "invalid ack wait",
			config: OrderedPublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				AckWait:           0,
			},
			expectError: true,
			errorMsg:    "ack wait must be positive",
		},
		{
			name: "invalid max pending",
			config: OrderedPublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				AckWait:           time.Second,
				MaxPending:        0,
			},
			expectError: true,
			errorMsg:    "max pending must be positive",
		},
		{
			name: "invalid retry attempts",
			config: OrderedPublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				AckWait:           time.Second,
				MaxPending:        100,
				RetryAttempts:     0,
			},
			expectError: true,
			errorMsg:    "retry attempts must be positive",
		},
		{
			name: "invalid retry wait",
			config: OrderedPublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				AckWait:           time.Second,
				MaxPending:        100,
				RetryAttempts:     3,
				RetryWait:         0,
			},
			expectError: true,
			errorMsg:    "retry wait must be positive",
		},
		{
			name: "valid config",
			config: OrderedPublisherConfig{
				Name:              "test",
				MessageSerializer: suite.serializer,
				MessageRegistry:   suite.registry,
				Destination:       "test.dest",
				AckWait:           time.Second,
				MaxPending:        100,
				RetryAttempts:     3,
				RetryWait:         time.Second,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.config.Validate()
			if tc.expectError {
				suite.Error(err)
				suite.Contains(err.Error(), tc.errorMsg)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ConfigTestSuite) TestOrderedToPublisherConfig() {
	ordered := OrderedPublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       "test.dest",
		DestinationPrefix: "test.prefix",
		AckWait:           time.Second,
		MaxPending:        100,
		RetryAttempts:     3,
		RetryWait:         time.Second,
	}

	publisherConfig := ordered.toPublisherConfig()

	suite.Equal(ordered.Name, publisherConfig.Name)
	suite.Equal(ordered.MessageSerializer, publisherConfig.MessageSerializer)
	suite.Equal(ordered.MessageRegistry, publisherConfig.MessageRegistry)
	suite.Equal(ordered.Destination, publisherConfig.Destination)
	suite.Equal(ordered.DestinationPrefix, publisherConfig.DestinationPrefix)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
