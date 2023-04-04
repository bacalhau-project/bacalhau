//go:build unit || !integration

package opts

import (
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		expected model.StorageSpec
		error    bool
	}{
		{
			name:  "ipfs",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/inputs",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:  "ipfs with path",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA:/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/mount/path",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:  "ipfs with explicit dst path",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,dst=/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/mount/path",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:  "ipfs with explicit src and dst",
			input: "src=ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,dst=/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/mount/path",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:  "ipfs with explicit dst overrides",
			input: "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA:/input,dst=/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
				Path:          "/mount/path",
				CID:           "QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA",
			},
		},
		{
			name:  "s3",
			input: "s3://myBucket/dir/file-001.txt",
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
			name:  "s3 with endpoint and region",
			input: "s3://myBucket/dir/file-001.txt,opt=endpoint=http://localhost:9000,opt=region=us-east-1",
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
			name:  "s3 with multiple colons",
			input: "s3://myBucket/dir:file:001.txt:/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceS3,
				Name:          "s3://myBucket/dir:file:001.txt",
				Path:          "/mount/path",
				S3: &model.S3StorageSpec{
					Bucket: "myBucket",
					Key:    "dir:file:001.txt",
				},
			},
		},
		{
			name:  "http with port",
			input: "https://example.com:9000/file:/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceURLDownload,
				Name:          "https://example.com:9000/file",
				Path:          "/mount/path",
				URL:           "https://example.com:9000/file",
			},
		},
		{
			name:  "http with port and explicit format",
			input: "src=https://example.com:9000/file,dst=/mount/path",
			expected: model.StorageSpec{
				StorageSource: model.StorageSourceURLDownload,
				Name:          "https://example.com:9000/file",
				Path:          "/mount/path",
				URL:           "https://example.com:9000/file",
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
			opt := StorageOpt{}
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

func TestParseMultipleInputs(t *testing.T) {
	opt := StorageOpt{}
	require.NoError(t, opt.Set("ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA"))
	require.NoError(t, opt.Set("s3://myBucket/dir/file-001.txt"))
	assert.Equal(t, 2, len(opt.Values()))
	assert.Equal(t, model.StorageSourceIPFS, opt.Values()[0].StorageSource)
	assert.Equal(t, model.StorageSourceS3, opt.Values()[1].StorageSource)
	assert.Equal(t, 2, len(strings.Split(opt.String(), ",")))
}
