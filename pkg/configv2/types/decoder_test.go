//go:build unit || !integration

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodePublisherConfig(t *testing.T) {
	// Define example configuration data
	configData := PublishersConfig{
		Disabled: []string{"someotherkind"},
		Config: map[string]map[string]interface{}{
			KindPublisherLocal: {
				"Address":   "127.0.0.1",
				"Port":      8080,
				"Directory": "/tmp/data",
			},
		},
	}

	// Expected LocalPublisherConfig instance
	expected := LocalPublisherConfig{
		Address:   "127.0.0.1",
		Port:      8080,
		Directory: "/tmp/data",
	}

	// Test the decoding function
	localConfig, err := DecodeProviderConfig[LocalPublisherConfig](configData)

	// Assertions
	assert.NoError(t, err, "Expected no error during decoding")
	assert.Equal(t, expected, localConfig, "Decoded config should match the expected struct")
}
