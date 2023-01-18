//go:build unit || !integration

package system

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemConfigSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemConfigSuite(t *testing.T) {
	suite.Run(t, new(SystemConfigSuite))
}

func (s *SystemConfigSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *SystemConfigSuite) TestMessageSigning() {
	defer func() {
		if r := recover(); r != nil {
			s.T().Errorf("unexpected panic: %v", r)
		}
	}()

	require.NoError(s.T(), InitConfigForTesting(s.T()))

	msg := []byte("Hello, world!")
	sig, err := SignForClient(msg)
	require.NoError(s.T(), err)

	ok, err := VerifyForClient(msg, sig)
	require.NoError(s.T(), err)
	require.True(s.T(), ok)

	publicKey := GetClientPublicKey()
	err = Verify(msg, sig, publicKey)
	require.NoError(s.T(), err)
}

func (s *SystemConfigSuite) TestGetClientID() {
	defer func() {
		if r := recover(); r != nil {
			s.T().Errorf("unexpected panic: %v", r)
		}
	}()

	var firstId string
	s.Run("first", func() {
		s.Require().NoError(InitConfigForTesting(s.T()))
		firstId = GetClientID()
		s.Require().NotEmpty(firstId)
	})

	var secondId string
	s.Run("second", func() {
		s.Require().NoError(InitConfigForTesting(s.T()))
		secondId = GetClientID()
		s.Require().NotEmpty(secondId)

		// Two different clients should have different IDs.
		s.Assert().NotEqual(firstId, secondId)
	})
}

func (s *SystemConfigSuite) TestPublicKeyMatchesID() {
	require.NoError(s.T(), InitConfigForTesting(s.T()))

	id := GetClientID()
	publicKey := GetClientPublicKey()
	ok, err := PublicKeyMatchesID(publicKey, id)
	require.NoError(s.T(), err)
	require.True(s.T(), ok)
}
