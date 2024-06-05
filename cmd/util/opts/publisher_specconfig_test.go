//go:build unit || !integration

package opts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestParsePublisherSpecConfig(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		expected *models.SpecConfig
		error    bool
	}{
		{
			name:  "ipfs",
			input: "ipfs",
			expected: &models.SpecConfig{
				Type:   models.PublisherIPFS,
				Params: make(map[string]interface{}),
			},
		},
		{
			name:  "s3",
			input: "s3://myBucket/dir/file-001.txt",
			expected: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "myBucket",
					"Key":      "dir/file-001.txt",
					"Endpoint": "",
					"Region":   "",
				},
			},
		},
		{
			name:  "s3 with endpoint and region",
			input: "s3://myBucket/dir/file-001.txt,opt=endpoint=http://127.0.0.1:9000,opt=region=us-east-1",
			expected: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "myBucket",
					"Key":      "dir/file-001.txt",
					"Endpoint": "http://127.0.0.1:9000",
					"Region":   "us-east-1",
				},
			},
		},
		{
			name:  "s3 non URI",
			input: "s3,opt=bucket=myBucket,opt=key=dir/file-001.txt,opt=region=us-east-1",
			expected: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "myBucket",
					"Key":      "dir/file-001.txt",
					"Region":   "us-east-1",
					"Endpoint": "",
				},
			},
		},
		{
			name:  "empty",
			input: "",
			error: true,
		},
		{
			name:  "invalid flags",
			input: "s3://myBucket/dir/file-001.txt,y=/mount/path",
			error: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			opt := PublisherSpecConfigOpt{}
			err := opt.Set(test.input)
			if test.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, opt.Value())
			}
		})
	}
}
