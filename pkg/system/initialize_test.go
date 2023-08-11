//go:build unit || !integration

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
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

	SetupBacalhauRepoForTesting(s.T())

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
		SetupBacalhauRepoForTesting(s.T())
		firstId = GetClientID()
		s.Require().NotEmpty(firstId)
	})

	var secondId string
	s.Run("second", func() {
		SetupBacalhauRepoForTesting(s.T())
		secondId = GetClientID()
		s.Require().NotEmpty(secondId)

		// Two different clients should have different IDs.
		s.Assert().NotEqual(firstId, secondId)
	})
}

func (s *SystemConfigSuite) TestPublicKeyMatchesID() {
	SetupBacalhauRepoForTesting(s.T())

	id := GetClientID()
	publicKey := GetClientPublicKey()
	ok, err := PublicKeyMatchesID(publicKey, id)
	s.NoError(err)
	s.True(ok)
}

// TODO(forrest): [fixme] I am removing this test because it creates a file in the home directory, tests should
// _______NEVER_______ do that
func (s *SystemConfigSuite) TestEnsureConfigDir() {
	s.T().Skip("skipping because test is creating a directory in home dir")
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
		{"/BADDIR", tempDir, tempDir},
	}
	for _, test := range tests {
		s.Run(
			fmt.Sprintf("root_dir_%t/bacalhau_dir_%t", test.root_dir != "", test.bacalhau_dir != ""),
			func() {
				s.T().Setenv("ROOT_DIR", test.root_dir)
				s.T().Setenv("BACALHAU_DIR", test.bacalhau_dir)
				configDir, err := SetupBacalhauRepo()
				s.DirExists(configDir)
				s.Equal(configDir, test.exp)
				s.NoError(err)
			})
	}
}

func (s *SystemConfigSuite) TestNiceErrorOnBadConfigDir() {
	badDirString := "/BADDIR"
	s.T().Setenv("BACALHAU_DIR", badDirString)
	_, err := SetupBacalhauRepo()
	s.Error(err)
	s.Contains(err.Error(), badDirString)
}
