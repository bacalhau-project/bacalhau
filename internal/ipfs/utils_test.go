package ipfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type IpfsUtilsTestSuite struct {
	suite.Suite
}

func (suite *IpfsUtilsTestSuite) SetupSuite() {
}

func (suite *IpfsUtilsTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
}

// Default hello world for bacalhau - execute with no arguments
func (suite *IpfsUtilsTestSuite) Test_IpfsNotSnap() {

	ipfs_binary, err := exec.LookPath("ipfs")

	if err != nil {
		assert.Fail(suite.T(), "Could not find 'ipfs' binary on your path.")
	}

	ipfs_binary_full_path, _ := filepath.Abs(ipfs_binary)

	if  !strings.Contains(ipfs_binary_full_path, "/snap/") {
		suite.Suite.T().Skip("ipfs not installed with snap, skipping test")
	}

	_, err = IpfsCommand("", nil)
	assert.Contains(suite.T(), string(err.Error()), "using snap")

}

func TestIpfsUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(IpfsUtilsTestSuite))
}
