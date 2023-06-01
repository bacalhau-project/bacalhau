//go:build unit || !integration

package job

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	storagetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStorageString(t *testing.T) {
	for _, test := range []struct {
		name        string
		source      string
		destination string
		options     map[string]string
		expected    spec.Storage
		error       bool
	}{
		{
			name:   "ipfs",
			source: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			expected: storagetesting.MakeIpfsStorageSpec(t,
				"ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				"/inputs",
				"QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			),
		},
		{
			name:        "ipfs with explicit dst path",
			source:      "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			destination: "/mount/path",
			expected: storagetesting.MakeIpfsStorageSpec(t,
				"ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				"/mount/path",
				"QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			),
		},
		{
			name:   "s3",
			source: "s3://myBucket/dir/file-001.txt",
			expected: storagetesting.MakeS3StorageSpec(t,
				"s3://myBucket/dir/file-001.txt",
				"/inputs",
				&spec_s3.S3StorageSpec{
					Bucket: "myBucket",
					Key:    "dir/file-001.txt",
				},
			),
		},
		{
			name:   "s3 with endpoint and region",
			source: "s3://myBucket/dir/file-001.txt",
			options: map[string]string{
				"endpoint": "http://localhost:9000",
				"region":   "us-east-1",
			},
			expected: storagetesting.MakeS3StorageSpec(t,
				"s3://myBucket/dir/file-001.txt",
				"/inputs",
				&spec_s3.S3StorageSpec{
					Bucket:   "myBucket",
					Key:      "dir/file-001.txt",
					Endpoint: "http://localhost:9000",
					Region:   "us-east-1",
				},
			),
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

func TestParsePublisherString(t *testing.T) {
	for _, test := range []struct {
		name         string
		publisherURI string
		options      map[string]interface{}
		expected     model.PublisherSpec
		error        bool
	}{
		{
			name:         "ipfs",
			publisherURI: "ipfs",
			expected: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		{
			name:         "ipfs as scheme",
			publisherURI: "ipfs://",
			expected: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		{
			name:         "lotus",
			publisherURI: "lotus",
			expected: model.PublisherSpec{
				Type: model.PublisherFilecoin,
			},
		},
		{
			name:         "estuary",
			publisherURI: "estuary",
			expected: model.PublisherSpec{
				Type: model.PublisherEstuary,
			},
		},
		{
			name:         "s3",
			publisherURI: "s3://myBucket/dir/file-001.txt",
			expected: model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket": "myBucket",
					"key":    "dir/file-001.txt",
				},
			},
		},
		{
			name:         "s3 with endpoint and region",
			publisherURI: "s3://myBucket/dir/file-001.txt",
			options: map[string]interface{}{
				"endpoint": "http://localhost:9000",
				"region":   "us-east-1",
				"archive":  true,
			},
			expected: model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket":   "myBucket",
					"key":      "dir/file-001.txt",
					"endpoint": "http://localhost:9000",
					"region":   "us-east-1",
					"archive":  true,
				},
			},
		},
		{
			name:         "empty",
			publisherURI: "",
			error:        true,
		},
		{
			name:         "invalid schema",
			publisherURI: "metalloca://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			error:        true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.options == nil {
				test.options = map[string]interface{}{}
			}
			spec, err := ParsePublisherString(test.publisherURI, test.options)
			if test.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, spec)
			}
		})
	}
}
