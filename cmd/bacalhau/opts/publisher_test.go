//go:build unit || !integration

package opts

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePublisher(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		expected model.PublisherSpec
		error    bool
	}{
		{
			name:  "ipfs",
			input: "ipfs",
			expected: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		{
			name:  "s3",
			input: "s3://myBucket/dir/file-001.txt",
			expected: model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket": "myBucket",
					"key":    "dir/file-001.txt",
				},
			},
		},
		{
			name:  "s3 with endpoint and region",
			input: "s3://myBucket/dir/file-001.txt,opt=endpoint=http://localhost:9000,opt=region=us-east-1,opt=compress=true",
			expected: model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket":   "myBucket",
					"key":      "dir/file-001.txt",
					"endpoint": "http://localhost:9000",
					"region":   "us-east-1",
					"compress": "true",
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
			input: "x=ipfs://QmXJ3wT1C27W8Vvc21NjLEb7VdNk9oM8zJYtDkG1yH2fnA,y=/mount/path",
			error: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			opt := PublisherOpt{}
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
