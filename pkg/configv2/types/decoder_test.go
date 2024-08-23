//go:build unit || !integration

package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestDecodePublisherConfig(t *testing.T) {
	// Define example configuration data
	configData := types.PublishersConfig{
		Disabled: []string{"someotherkind"},
		Config: map[string]map[string]interface{}{
			models.PublisherLocal: {
				"Address":   "127.0.0.1",
				"Port":      8080,
				"Directory": "/tmp/data",
			},
		},
	}

	// Expected LocalPublisherConfig instance
	expected := types.LocalPublisherConfig{
		Address:   "127.0.0.1",
		Port:      8080,
		Directory: "/tmp/data",
	}

	// Test the decoding function
	localConfig, err := types.DecodeProviderConfig[types.LocalPublisherConfig](configData)

	// Assertions
	assert.NoError(t, err, "Expected no error during decoding")
	assert.Equal(t, expected, localConfig, "Decoded config should match the expected struct")
}
