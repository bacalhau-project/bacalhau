//go:build unit || !integration

package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SetupSuite struct {
	suite.Suite
}

func TestSetupSuite(t *testing.T) {
	suite.Run(t, new(SetupSuite))
}

func (s *SetupSuite) TestEnsureConfigDir() {
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
				configDir, err := SetupBacalhauRepo("")
				s.DirExists(configDir)
				s.Equal(configDir, test.exp)
				s.NoError(err)
			})
	}
}

func (s *SetupSuite) TestNiceErrorOnBadConfigDir() {
	badDirString := "/BADDIR"
	s.T().Setenv("BACALHAU_DIR", badDirString)
	_, err := SetupBacalhauRepo("")
	s.Error(err)
	s.Contains(err.Error(), badDirString)
}
