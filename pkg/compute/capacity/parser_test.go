package capacity

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestResourceUsageConfigParser(t *testing.T) {
	testCases := []struct {
		usageConfig  model.ResourceUsageConfig
		expectedData model.ResourceUsageData
	}{
		{
			model.ResourceUsageConfig{
				CPU:    "100m",
				Memory: "100Mi",
				Disk:   "",
				GPU:    "",
			},
			model.ResourceUsageData{
				CPU:    0.1,               // 100m
				Memory: 100 * 1024 * 1024, // 100Mi
				Disk:   0,
				GPU:    0,
			},
		},
	}

	for _, tc := range testCases {
		data := ParseResourceUsageConfig(tc.usageConfig)
		require.Equal(t, tc.expectedData, data)
	}
}
