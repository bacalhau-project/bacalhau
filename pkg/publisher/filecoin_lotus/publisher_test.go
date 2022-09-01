package filecoinlotus

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var ctx context.Context
var tempDir string
var driver *FilecoinLotusPublisher
var cm *system.CleanupManager

const TestJobId = "job-123"
const TestHostId = "host-123"
const TestContentCid = "QmQEDtn7tSFxgquj5ZHdFVKimSb14w1bmnbGyRQ5ukQLcF"
const TestDealCid = "bafyreict2zhkbwy2arri3jgthk2jyznck47umvpqis3hc5oclvskwpteau"
const TestMinerAddress = "f01000"
const TestStoragePrice = "5"
const TestStorageDuration = "100"

type FilecoinPublisherSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestFilecoinPublisherSuite(t *testing.T) {
	suite.Run(t, new(FilecoinPublisherSuite))
}

// Before all suite
func (suite *FilecoinPublisherSuite) SetupAllSuite() {

}

// Before each test
func (suite *FilecoinPublisherSuite) SetupTest() {
	var setupErr error
	cm = system.NewCleanupManager()
	ctx = context.Background()
	resolver := job.NewStateResolver(
		func(ctx context.Context, id string) (model.Job, error) {
			return model.Job{}, nil
		},
		func(ctx context.Context, id string) (model.JobState, error) {
			return model.JobState{}, nil
		},
	)
	tempDir, setupErr = ioutil.TempDir("", "bacalhau-filecoin-lotus-test")
	require.NoError(suite.T(), setupErr)
	os.Setenv("LOTUS_LOGFILE", fmt.Sprintf("%s/logs.txt", tempDir))
	os.Setenv("LOTUS_TEST_CONTENT_CID", TestContentCid)
	os.Setenv("LOTUS_TEST_DEAL_CID", TestDealCid)
	driver, setupErr = NewFilecoinLotusPublisher(cm, resolver, FilecoinLotusPublisherConfig{
		ExecutablePath:  "../../../testdata/mocks/lotus.sh",
		MinerAddress:    TestMinerAddress,
		StoragePrice:    TestStoragePrice,
		StorageDuration: TestStorageDuration,
	})
	require.NoError(suite.T(), setupErr)
}

func (suite *FilecoinPublisherSuite) TearDownTest() {
}

func (suite *FilecoinPublisherSuite) TearDownAllSuite() {

}

func (suite *FilecoinPublisherSuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
	dat, err := os.ReadFile(fmt.Sprintf("%s/logs.txt", tempDir))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "command: version\n0.0.1\n", string(dat))
}

func (suite *FilecoinPublisherSuite) TestPublishShardResult() {
	resultsDir, err := ioutil.TempDir("", "bacalhau-filecoin-lotus-test")
	require.NoError(suite.T(), err)
	err = os.WriteFile(fmt.Sprintf("%s/file.txt", resultsDir), []byte("hello"), 0644)
	require.NoError(suite.T(), err)
	publishResult, err := driver.PublishShardResult(ctx, model.JobShard{
		Job: model.Job{
			ID: TestJobId,
		},
	}, TestHostId, resultsDir)
	require.NoError(suite.T(), err)

	commandLogs, err := os.ReadFile(fmt.Sprintf("%s/logs.txt", tempDir))
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), fmt.Sprintf("job-%s-shard-%d-host-%s", TestJobId, 0, TestHostId), publishResult.Name)
	require.Equal(suite.T(), TestContentCid, publishResult.Cid)
	require.Equal(suite.T(), model.StorageSourceFilecoin, publishResult.Engine)
	require.NotNil(suite.T(), publishResult.Metadata)
	require.Equal(suite.T(), 1, len(publishResult.Metadata))
	dealCid, ok := publishResult.Metadata["deal_cid"]
	require.True(suite.T(), ok)
	require.Equal(suite.T(), TestDealCid, dealCid)

	logLines := strings.Split(string(commandLogs), "\n")
	firstLine := logLines[0]
	logLines = logLines[1:]

	require.True(suite.T(), strings.Contains(firstLine, "command: client import /tmp/bacalhau-filecoin-lotus"))

	expectedLogs := fmt.Sprintf(`Import 3, Root %s
command: client deal %s %s %s %s
.. executing
Deal (%s) CID: %s
`,
		TestContentCid,
		TestContentCid,
		TestMinerAddress,
		TestStoragePrice,
		TestStorageDuration,
		TestMinerAddress,
		TestDealCid,
	)

	require.Equal(suite.T(), expectedLogs, strings.Join(logLines, "\n"))
}
