//go:build unit || !integration

package s3

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (s *ParamsTestSuite) TestDecodeMap() {
	expected := PublisherSpec{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "localhost:9000",
		Region:   "us-east-1",
		Encoding: EncodingPlain,
	}
	decoded, err := DecodePublisherSpec(&models.SpecConfig{
		Type:   models.PublisherS3,
		Params: expected.ToMap(),
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *ParamsTestSuite) TestDecodeInterface() {
	expected := PublisherSpec{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "localhost:9000",
		Region:   "us-east-1",
		Encoding: EncodingPlain,
	}
	params := map[string]interface{}{
		"Bucket":   expected.Bucket,
		"Key":      expected.Key,
		"Endpoint": expected.Endpoint,
		"Region":   expected.Region,
		"Encoding": string(EncodingPlain),
	}
	decoded, err := DecodePublisherSpec(&models.SpecConfig{
		Type:   models.PublisherS3,
		Params: params,
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *ParamsTestSuite) TestDecodeInterfaceLowerCase() {
	expected := PublisherSpec{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "localhost:9000",
		Region:   "us-east-1",
		Encoding: EncodingGzip,
	}
	params := map[string]interface{}{
		"bucket":   expected.Bucket,
		"key":      expected.Key,
		"endpoint": expected.Endpoint,
		"region":   expected.Region,
		"encoding": string(EncodingGzip),
	}
	decoded, err := DecodePublisherSpec(&models.SpecConfig{
		Type:   models.PublisherS3,
		Params: params,
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *ParamsTestSuite) TestDecodeJson() {
	expected := PublisherSpec{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "localhost:9000",
		Region:   "us-east-1",
		Encoding: EncodingPlain,
	}
	bytes, err := json.Marshal(expected)
	s.Require().NoError(err)

	var unmarshalled map[string]interface{}
	err = json.Unmarshal(bytes, &unmarshalled)
	s.Require().NoError(err)

	if err != nil {
		return
	}
	decoded, err := DecodePublisherSpec(&models.SpecConfig{
		Type:   models.PublisherS3,
		Params: unmarshalled,
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *ParamsTestSuite) TestDecodeInvalidType() {
	_, err := DecodePublisherSpec(&models.SpecConfig{
		Type: models.PublisherIPFS,
	})
	s.Require().Error(err)
}

func (s *ParamsTestSuite) TestDecodeInvalidParams() {
	for _, tc := range []struct {
		name string
		spec *models.SpecConfig
	}{
		{
			name: "empty bucket",
			spec: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket": "",
					"Key":    uuid.NewString(),
				},
			},
		},
		{
			name: "empty key",
			spec: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket": "bucket",
					"Key":    "",
				},
			},
		},
		{
			name: "invalid encoding",
			spec: &models.SpecConfig{
				Type: models.PublisherS3,
				Params: map[string]interface{}{
					"Bucket":   "bucket",
					"Key":      uuid.NewString(),
					"Encoding": "invalid",
				},
			},
		},
	} {
		s.Run(tc.name, func() {
			_, err := DecodePublisherSpec(tc.spec)
			s.Require().Error(err)
		})
	}
}
