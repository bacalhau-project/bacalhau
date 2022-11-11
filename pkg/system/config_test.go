//go:build !integration

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

// Before each test
func (suite *SystemConfigSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemConfigSuite) TestMessageSigning() {
	defer func() {
		if r := recover(); r != nil {
			suite.T().Errorf("unexpected panic: %v", r)
		}
	}()

	require.NoError(suite.T(), InitConfigForTesting())

	msg := []byte("Hello, world!")
	sig, err := SignForClient(msg)
	require.NoError(suite.T(), err)

	ok, err := VerifyForClient(msg, sig)
	require.NoError(suite.T(), err)
	require.True(suite.T(), ok)

	publicKey := GetClientPublicKey()
	err = Verify(msg, sig, publicKey)
	require.NoError(suite.T(), err)
}

func (suite *SystemConfigSuite) TestGetClientID() {
	defer func() {
		if r := recover(); r != nil {
			suite.T().Errorf("unexpected panic: %v", r)
		}
	}()

	require.NoError(suite.T(), InitConfigForTesting())
	id := GetClientID()
	require.NotEmpty(suite.T(), id)

	require.NoError(suite.T(), InitConfigForTesting())
	id2 := GetClientID()
	require.NotEmpty(suite.T(), id2)

	// Two different clients should have different IDs.
	require.NotEqual(suite.T(), id, id2)
}

func (suite *SystemConfigSuite) TestPublicKeyMatchesID() {
	defer func() {
		if r := recover(); r != nil {
			suite.T().Errorf("unexpected panic: %v", r)
		}
	}()

	require.NoError(suite.T(), InitConfigForTesting())

	id := GetClientID()
	publicKey := GetClientPublicKey()
	ok, err := PublicKeyMatchesID(publicKey, id)
	require.NoError(suite.T(), err)
	require.True(suite.T(), ok)
}
