//go:build unit || !integration

package orchestrator

import (
	"testing"

	"github.com/Masterminds/semver"

	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func TestListResponseRequiresMigration(t *testing.T) {
	tests := []struct {
		name     string
		version  *semver.Version
		expected bool
	}{
		{
			name:     "version less than 1.3.2",
			version:  semver.MustParse("1.2.0"),
			expected: true,
		},
		{
			name:     "version equal to 1.3.2",
			version:  version.V1_3_2,
			expected: true,
		},
		{
			name:     "development version",
			version:  version.Development,
			expected: false,
		},
		{
			name:     "unknown version",
			version:  version.Unknown,
			expected: false,
		},
		{
			name:     "version greater than 1.3.2",
			version:  semver.MustParse("1.4.0"),
			expected: false,
		},
		{
			name:     "version equal to 1.3.2 with prerelease",
			version:  semver.MustParse("v1.3.2-alpha"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := listResponseRequiresMigration(tt.version)
			if result != tt.expected {
				t.Errorf("listResponseRequiresMigration(%v) = %v; want %v", tt.version, result, tt.expected)
			}
		})
	}
}
