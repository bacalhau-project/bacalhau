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
		expected *model.PublisherSpec
		error    bool
	}{
		{
			name:  "ipfs",
			input: "ipfs",
			expected: &model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		{
			name:  "s3",
			input: "s3://myBucket/dir/file-001.txt",
			expected: &model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket": "myBucket",
					"key":    "dir/file-001.txt",
				},
			},
		},
		{
			name:  "s3 with endpoint and region",
			input: "s3://myBucket/dir/file-001.txt,opt=endpoint=http://127.0.0.1:9000,opt=region=us-east-1",
			expected: &model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket":   "myBucket",
					"key":      "dir/file-001.txt",
					"endpoint": "http://127.0.0.1:9000",
					"region":   "us-east-1",
				},
			},
		},
		{
			name:  "s3 non URI",
			input: "s3,opt=bucket=myBucket,opt=key=dir/file-001.txt,opt=region=us-east-1",
			expected: &model.PublisherSpec{
				Type: model.PublisherS3,
				Params: map[string]interface{}{
					"bucket": "myBucket",
					"key":    "dir/file-001.txt",
					"region": "us-east-1",
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
