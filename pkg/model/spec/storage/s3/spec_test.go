package s3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
)

func TestRoundTrip(t *testing.T) {
	expectedSpec := s3.S3StorageSpec{
		Bucket:         "bucket",
		Key:            "key",
		ChecksumSHA256: "checksum",
		VersionID:      "versionID",
		Endpoint:       "endpoint",
		Region:         "region",
	}

	spec, err := expectedSpec.AsSpec()
	require.NoError(t, err)

	require.NotEmpty(t, spec.SchemaData)
	require.NotEmpty(t, spec.Params)

	require.True(t, s3.Schema.Cid().Equals(spec.Schema))

	actualSpec, err := s3.Decode(spec)
	require.NoError(t, err)

	assert.Equal(t, expectedSpec.Bucket, actualSpec.Bucket)
	assert.Equal(t, expectedSpec.Key, actualSpec.Key)
	assert.Equal(t, expectedSpec.ChecksumSHA256, actualSpec.ChecksumSHA256)
	assert.Equal(t, expectedSpec.VersionID, actualSpec.VersionID)
	assert.Equal(t, expectedSpec.Endpoint, actualSpec.Endpoint)
	assert.Equal(t, expectedSpec.Region, actualSpec.Region)
}
