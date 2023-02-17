//go:build unit || !integration

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
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

	s.NoError(InitConfigForTesting(s.T()))

	msg := []byte("Hello, world!")
	sig, err := SignForClient(msg)
	s.NoError(err)

	ok, err := VerifyForClient(msg, sig)
	s.NoError(err)
	s.True(ok)

	publicKey := GetClientPublicKey()
	err = Verify(msg, sig, publicKey)
	s.NoError(err)
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
	s.NoError(InitConfigForTesting(s.T()))

	id := GetClientID()
	publicKey := GetClientPublicKey()
	ok, err := PublicKeyMatchesID(publicKey, id)
	s.NoError(err)
	s.True(ok)
}

func (s *SystemConfigSuite) TestEnsureConfigDir() {
	tempDir := s.T().TempDir()
	home, err := os.UserHomeDir()
	s.NoError(err)
	default_dir := filepath.Join(home, ".bacalhau")
	s.T().Setenv("FIL_WALLET_ADDRESS", "placeholder_address")
	tests := []struct {
		root_dir     string
		bacalhau_dir string
		exp          string
	}{
		{"", "", default_dir},
		{tempDir, "", tempDir},
		{"", tempDir, tempDir},
		{tempDir, default_dir, default_dir},
	}
	for _, test := range tests {
		s.Run(
			fmt.Sprintf("root_dir_%t/bacalhau_dir_%t", test.root_dir != "", test.bacalhau_dir != ""),
			func() {
				s.T().Setenv("ROOT_DIR", test.root_dir)
				s.T().Setenv("BACALHAU_DIR", test.bacalhau_dir)
				configDir, err := ensureConfigDir()
				s.DirExists(configDir)
				s.Equal(configDir, test.exp)
				s.NoError(err)
			})
	}
}
