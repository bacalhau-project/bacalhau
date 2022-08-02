package verifier

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type VerifierIPFSSuite struct {
	suite.Suite
}

// a normal test function and pass our suite to suite.Run
func TestVerifierIPFSSuite(t *testing.T) {
	suite.Run(t, new(VerifierIPFSSuite))
}

// Before all suite
func (suite *VerifierIPFSSuite) SetupAllSuite() {

}

// Before each test
func (suite *VerifierIPFSSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *VerifierIPFSSuite) TearDownTest() {
}

func (suite *VerifierIPFSSuite) TearDownAllSuite() {

}

func (suite *VerifierIPFSSuite) TestIPFSVerifier() {
	// TODO: #388 This test seems broken due to existence of output directory
	suite.T().Skip("REMOVE_WHEN_OUTPUTDIRECTORY_QUESTION_ANSWERED https://github.com/filecoin-project/bacalhau/issues/388")

	ctx := context.Background()
	stack, cm := SetupTest(suite.T(), 1)
	defer TeardownTest(stack, cm)

	tempDirPrefix := "bacalhau-ipfs-verifier-test"

	inputDir, err := ioutil.TempDir("", tempDirPrefix)
	require.NoError(suite.T(), err)

	outputDir, err := ioutil.TempDir("", tempDirPrefix)
	require.NoError(suite.T(), err)

	fixtureContent := "hello world"
	err = os.WriteFile(inputDir+"/file.txt", []byte(fixtureContent), 0644)
	require.NoError(suite.T(), err)

	verifier, err := ipfs.NewVerifier(
		cm, stack.Nodes[0].IpfsClient.APIAddress())
	require.NoError(suite.T(), err)

	installed, err := verifier.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)

	resultHash, err := verifier.ProcessShardResultsFolder(ctx, "fake-job-id", 0, inputDir)
	require.NoError(suite.T(), err)

	outputPath := filepath.Join(outputDir, resultHash)
	err = verifier.IPFSClient.Get(ctx, resultHash, outputPath)
	require.NoError(suite.T(), err)

	outputContent, err := os.ReadFile(outputPath + "/file.txt")
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), fixtureContent, string(outputContent))
}
