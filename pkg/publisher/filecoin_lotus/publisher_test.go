package filecoinlotus

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FilecoinPublisherSuite struct {
	suite.Suite

	dataDir   string
	uploadDir string

	driver *Publisher
}

func TestFilecoinPublisherSuite(t *testing.T) {
	dataDir := os.Getenv("LOTUS_PATH")
	uploadDir := os.Getenv("LOTUS_UPLOAD_DIR")
	if dataDir == "" || uploadDir == "" {
		t.Skip("Skipping Lotus provider as it currently needs the Lotus to be started manually")
	}

	suite.Run(t, &FilecoinPublisherSuite{
		dataDir:   dataDir,
		uploadDir: uploadDir,
	})
}

func (s *FilecoinPublisherSuite) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())

	cm := system.NewCleanupManager()
	s.T().Cleanup(cm.Cleanup)

	resolver := job.NewStateResolver(
		func(ctx context.Context, id string) (*model.Job, error) {
			return &model.Job{}, nil
		},
		func(ctx context.Context, id string) (model.JobState, error) {
			return model.JobState{}, nil
		},
	)
	driver, setupErr := NewFilecoinLotusPublisher(context.Background(), cm, resolver, PublisherConfig{
		MinerAddress:    "t01000",
		StorageDuration: 24 * 24 * time.Hour,
		LotusDataDir:    s.dataDir,
		LotusUploadDir:  s.uploadDir,
	})
	require.NoError(s.T(), setupErr)

	s.driver = driver
}

func (s *FilecoinPublisherSuite) TestIsInstalled() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	installed, err := s.driver.IsInstalled(ctx)

	assert.NoError(s.T(), err)
	assert.True(s.T(), installed)
}

func (s *FilecoinPublisherSuite) TestPublishShardResult() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resultsDir := s.T().TempDir()
	require.NoError(s.T(), os.WriteFile(filepath.Join(resultsDir, "file.txt"), []byte("hello"), 0644))

	publishResult, err := s.driver.PublishShardResult(ctx, model.JobShard{
		Job: &model.Job{
			ID: "job-123",
		},
	}, "host-123", resultsDir)
	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), publishResult.Name)
	assert.NotEmpty(s.T(), publishResult.CID)
	require.Equal(s.T(), model.StorageSourceFilecoin, publishResult.StorageSource)
	require.NotNil(s.T(), publishResult.Metadata)
	require.Equal(s.T(), 1, len(publishResult.Metadata))
	dealCid := publishResult.Metadata["deal_cid"]
	assert.NotEmpty(s.T(), dealCid)

	// Need to re-read the file back out of Filecoin to verify it was saved successfully
}
