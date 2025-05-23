package node

import (
	"context"
	"runtime"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestRuntimeLabelsProvider(t *testing.T) {
	provider := NewRuntimeLabelsProvider()
	labels := provider.GetLabels(context.Background())

	// Check that the labels contain the expected keys
	assert.Contains(t, labels, LabelOperatingSystem)
	assert.Contains(t, labels, LabelArchitecture)

	// Check that the values match the runtime values
	assert.Equal(t, runtime.GOOS, labels[LabelOperatingSystem])
	assert.Equal(t, runtime.GOARCH, labels[LabelArchitecture])
}

func TestConfigLabelsProvider(t *testing.T) {
	staticLabels := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	provider := NewConfigLabelsProvider(staticLabels)
	labels := provider.GetLabels(context.Background())

	// Check that all static labels are present
	assert.Equal(t, staticLabels, labels)
}

func TestNameLabelsProvider(t *testing.T) {
	nodeName := "test-node"
	provider := NewNameLabelsProvider(nodeName)
	labels := provider.GetLabels(context.Background())

	// Check that the label contains the expected key
	assert.Contains(t, labels, LabelNodeName)

	// Check that the value matches the provided node name
	assert.Equal(t, nodeName, labels[LabelNodeName])
}

func TestLabelProvidersInterface(t *testing.T) {
	// Test that all providers implement the models.LabelsProvider interface
	var _ models.LabelsProvider = (*RuntimeLabelsProvider)(nil)
	var _ models.LabelsProvider = (*ConfigLabelsProvider)(nil)
	var _ models.LabelsProvider = (*NameLabelsProvider)(nil)
}
