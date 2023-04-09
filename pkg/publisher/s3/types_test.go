//go:build unit || !integration

package s3

import (
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type PublisherConfigTestSuite struct {
	suite.Suite
}

func TestPublisherConfigTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherConfigTestSuite))
}

func (s *PublisherConfigTestSuite) TestDecodeMap() {
	expected := PublisherConfig{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "endpoint",
		Region:   "region",
	}
	decoded, err := DecodeConfig(model.PublisherSpec{
		Type:   model.PublisherS3,
		Config: expected.ToMap(),
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *PublisherConfigTestSuite) TestDecodeJson() {
	expected := PublisherConfig{
		Bucket:   "bucket",
		Key:      uuid.NewString(),
		Endpoint: "endpoint",
		Region:   "region",
	}
	bytes, err := json.Marshal(expected)
	s.Require().NoError(err)

	var unmarshalled map[string]interface{}
	err = json.Unmarshal(bytes, &unmarshalled)
	s.Require().NoError(err)

	if err != nil {
		return
	}
	decoded, err := DecodeConfig(model.PublisherSpec{
		Type:   model.PublisherS3,
		Config: unmarshalled,
	})
	s.Require().NoError(err)
	s.Equal(expected, decoded)
}

func (s *PublisherConfigTestSuite) TestDecodeInvalidType() {
	_, err := DecodeConfig(model.PublisherSpec{
		Type: model.PublisherIpfs,
	})
	s.Require().Error(err)
}

func (s *PublisherConfigTestSuite) TestDecodeInvalidConfig() {
	for _, tc := range []struct {
		name string
		spec model.PublisherSpec
	}{
		{
			name: "empty bucket",
			spec: model.PublisherSpec{
				Type: model.PublisherS3,
				Config: map[string]interface{}{
					"Bucket": "",
					"Key":    uuid.NewString(),
				},
			},
		},
		{
			name: "empty key",
			spec: model.PublisherSpec{
				Type: model.PublisherS3,
				Config: map[string]interface{}{
					"Bucket": "bucket",
					"Key":    "",
				},
			},
		},
	} {
		s.Run(tc.name, func() {
			_, err := DecodeConfig(tc.spec)
			s.Require().Error(err)
		})
	}
}
