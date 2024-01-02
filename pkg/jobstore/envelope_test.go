//go:build unit || !integration

package jobstore

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/stretchr/testify/suite"
)

type EnvelopeTestSuite struct {
	suite.Suite
}

func TestEnvelopeTestSuite(t *testing.T) {
	suite.Run(t, new(EnvelopeTestSuite))
}

func (s *EnvelopeTestSuite) TestEnvelope() {
	type test struct {
		Value string
	}

	e := NewEnvelope[test](
		WithBody(test{Value: "hello"}),
		WithMarshaller[test](marshaller.NewJSONMarshaller()),
	)

	encoded, err := e.Serialize()
	s.NoError(err)

	newEnv, err := e.Deserialize(encoded)
	s.NoError(err)
	s.Equal(newEnv.Body.Value, e.Body.Value)
}

func (s *EnvelopeTestSuite) TestEnvelopeBinaryEncoding() {
	type test struct {
		Value string
	}

	e := NewEnvelope[test](
		WithBody(test{Value: "hello"}),
		WithMarshaller[test](marshaller.NewBinaryMarshaller()),
	)

	encoded, err := e.Serialize()
	s.NoError(err)

	newEnv, err := e.Deserialize(encoded)
	s.NoError(err)
	s.Equal(newEnv.Body.Value, e.Body.Value)
}
