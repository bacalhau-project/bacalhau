package filecoin_lotus

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
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
	driver, setupErr = NewFilecoinLotusPublisher(cm, resolver, FilecoinLotusPublisherConfig{
		ExecutablePath: "../../../testdata/mocks/lotus.sh",
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
