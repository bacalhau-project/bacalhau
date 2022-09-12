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
const TestMinerAddress = "t01000"
const TestStoragePrice = "0.000000000246842652"
const TestStorageDuration = "518577"

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
	os.Setenv("LOTUS_PATH", "/home/prash/.lotus-local-net")
	os.Setenv("LOTUS_MINER_PATH", "/home/prash/.lotus-miner-local-net")
	driver, setupErr = NewFilecoinLotusPublisher(cm, resolver, FilecoinLotusPublisherConfig{
		ExecutablePath:  "/home/prash/workspace/lotus-local-net/lotus",
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
}

func (suite *FilecoinPublisherSuite) TestListDeals() {
	deals, err := driver.listDeals(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), deals)
}

func (suite *FilecoinPublisherSuite) TestPublishShardResult() {
	tmpDirPrefix := "bacalhau-filecoin-lotus-test"
	resultsDir, err := ioutil.TempDir("", tmpDirPrefix)
	require.NoError(suite.T(), err)
	payloadPath := fmt.Sprintf("%s/payload.txt", resultsDir)
	err = ioutil.WriteFile(payloadPath, make([]byte, 1000), 0644)
	require.NoError(suite.T(), err)
	publishResult, err := driver.PublishShardResult(ctx, model.JobShard{
		Job: model.Job{
			ID: TestJobId,
		},
	}, TestHostId, payloadPath)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), fmt.Sprintf("job-%s-shard-%d-host-%s", TestJobId, 0, TestHostId), publishResult.Name)
	require.Equal(suite.T(), model.StorageSourceFilecoin, publishResult.Engine)
	require.NotNil(suite.T(), publishResult.Metadata)
	require.Equal(suite.T(), 1, len(publishResult.Metadata))
	dealCid, ok := publishResult.Metadata["deal_cid"]
	require.True(suite.T(), ok)
	require.NotNil(suite.T(), dealCid)
	
	deals, err := driver.listDeals(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), strings.Contains(deals, dealCid))
}
