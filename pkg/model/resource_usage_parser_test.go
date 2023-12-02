//go:build unit || !integration

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceUsageConfigParser(t *testing.T) {
	testCases := []struct {
		usageConfig  ResourceUsageConfig
		expectedData ResourceUsageData
	}{
		{
			ResourceUsageConfig{
				CPU:    "100m",
				Memory: "100Mi",
				Disk:   "",
				GPU:    "",
			},
			ResourceUsageData{
				CPU:    0.1,               // 100m
				Memory: 100 * 1024 * 1024, // 100Mi
				Disk:   0,
				GPU:    0,
			},
		},
	}

	for _, tc := range testCases {
		data, err := ParseResourceUsageConfig(tc.usageConfig)
		require.NoError(t, err)
		require.Equal(t, tc.expectedData, data)
	}
}
