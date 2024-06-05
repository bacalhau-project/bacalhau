//go:build unit || !integration

package opts

import (
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInputSourceSpecConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		expected *models.InputSource
		error    bool
	}{
		{
			name:  "ipfs",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
					},
				},
				Alias:  "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Target: "/inputs",
			},
		},
		{
			name:  "ipfs with path",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA:/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
					},
				},
				Alias:  "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Target: "/mount/path",
			},
		},
		{
			name:  "ipfs with explicit dst path",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,dst=/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
					},
				},
				Alias:  "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Target: "/mount/path",
			},
		},
		{
			name:  "ipfs with explicit src and dst",
			input: "src=ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,dst=/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
					},
				},
				Alias:  "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Target: "/mount/path",
			},
		},
		{
			name:  "ipfs with explicit dst overrides",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA:/input,dst=/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceIPFS,
					Params: map[string]interface{}{
						"CID": "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
					},
				},
				Alias:  "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Target: "/mount/path",
			},
		},
		{
			name:  "s3",
			input: "s3://myBucket/dir/file-001.txt",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: map[string]interface{}{
						"Bucket":         "myBucket",
						"Key":            "dir/file-001.txt",
						"Region":         "",
						"Endpoint":       "",
						"ChecksumSHA256": "",
						"VersionID":      "",
						"Filter":         "",
					},
				},
				Alias:  "s3://myBucket/dir/file-001.txt",
				Target: "/inputs",
			},
		},
		{
			name:  "s3 with endpoint and region",
			input: "s3://myBucket/dir/file-001.txt,opt=endpoint=http://127.0.0.1:9000,opt=region=us-east-1",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: map[string]interface{}{
						"Bucket":         "myBucket",
						"Key":            "dir/file-001.txt",
						"Endpoint":       "http://127.0.0.1:9000",
						"Region":         "us-east-1",
						"ChecksumSHA256": "",
						"VersionID":      "",
						"Filter":         "",
					},
				},
				Alias:  "s3://myBucket/dir/file-001.txt",
				Target: "/inputs",
			},
		},
		{
			name:  "s3 with multiple colons",
			input: "s3://myBucket/dir:file:001.txt:/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceS3,
					Params: map[string]interface{}{
						"Bucket":         "myBucket",
						"Key":            "dir:file:001.txt",
						"Endpoint":       "",
						"Region":         "",
						"ChecksumSHA256": "",
						"VersionID":      "",
						"Filter":         "",
					},
				},
				Alias:  "s3://myBucket/dir:file:001.txt",
				Target: "/mount/path",
			},
		},
		{
			name:  "http with port",
			input: "https://example.com:9000/file:/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceURL,
					Params: map[string]interface{}{
						"URL": "https://example.com:9000/file",
					},
				},
				Alias:  "https://example.com:9000/file",
				Target: "/mount/path",
			},
		},
		{
			name:  "http with port and explicit format",
			input: "src=https://example.com:9000/file,dst=/mount/path",
			expected: &models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceURL,
					Params: map[string]interface{}{
						"URL": "https://example.com:9000/file",
					},
				},
				Alias:  "https://example.com:9000/file",
				Target: "/mount/path",
			},
		},
		{
			name:  "empty",
			input: "",
			error: true,
		},
		{
			name:  "invalid flags",
			input: "x=ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,y=/mount/path",
			error: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			opt := StorageSpecConfigOpt{}
			err := opt.Set(test.input)
			if test.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, opt.Values()[0])
			}
		})
	}
}

func TestParseMultipleStorageInputSources(t *testing.T) {
	opt := StorageSpecConfigOpt{}
	require.NoError(t, opt.Set("ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA"))
	require.NoError(t, opt.Set("s3://myBucket/dir/file-001.txt"))
	assert.Equal(t, 2, len(opt.Values()))
	assert.Equal(t, model.StorageSourceIPFS, opt.Values()[0].Source.Type)
	assert.Equal(t, model.StorageSourceS3, opt.Values()[1].Source.Type)
	assert.Equal(t, 2, len(strings.Split(opt.String(), ",")))
}
