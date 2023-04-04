//go:build unit || !integration

package job

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStorageString(t *testing.T) {
	for _, test := range []struct {
		name        string
		source      string
		destination string
		options     map[string]string
		expected    model.StorageSpec
		error       bool
	}{
		{
			name:   "ipfs",
			source: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/inputs",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:        "ipfs with explicit dst path",
			source:      "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			destination: "/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/mount/path",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:   "s3",
			source: "s3://myBucket/dir/file-001.txt",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceS3,
				Name:          "s3://myBucket/dir/file-001.txt",
				Path:          "/inputs",
				S3: &model.S3StorageSpec{
					Bucket: "myBucket",
					Key:    "dir/file-001.txt",
				},
			},
		},
		{
			name:   "s3 with endpoint and region",
			source: "s3://myBucket/dir/file-001.txt",
			options: map[string]string{
				"endpoint": "http://localhost:9000",
				"region":   "us-east-1",
			},
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceS3,
				Name:          "s3://myBucket/dir/file-001.txt",
				Path:          "/inputs",
				S3: &model.S3StorageSpec{
					Bucket:   "myBucket",
					Key:      "dir/file-001.txt",
					Endpoint: "http://localhost:9000",
					Region:   "us-east-1",
				},
			},
		},
		{
			name:   "empty",
			source: "",
			error:  true,
		},
		{
			name:   "invalid schema",
			source: "metalloca://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			error:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec, err := ParseStorageString(test.source, test.destination, test.options)
			if test.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, spec)
			}
		})
	}
}
