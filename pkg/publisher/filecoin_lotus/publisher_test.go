package filecoinlotus

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus/api"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func init() {
	// make sure system.GetRandomString returns strings that are different every time - stop Lotus repeatedly creating
	// deals for the same content
	rand.Seed(time.Now().UnixNano())
}

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
	logger.ConfigureTestLogging(s.T())
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
		StorageDuration: 24 * 24 * time.Hour,
		LotusDataDir:    s.dataDir,
		LotusUploadDir:  s.uploadDir,
		MaximumPing:     2 * time.Second,
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

	expectedContent := system.GetRandomString(1000)

	resultsDir := s.T().TempDir()
	require.NoError(s.T(), os.WriteFile(filepath.Join(resultsDir, "file.txt"), []byte(expectedContent), 0644))

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
	assert.NotEmpty(s.T(), publishResult.Metadata["deal_cid"])

	imp := s.retrieveUploadedFile(ctx, publishResult.CID)

	output, err := os.MkdirTemp(s.driver.config.LotusUploadDir, "")
	require.NoError(s.T(), err)

	output = filepath.Join(output, "download")

	require.NoError(s.T(), s.driver.client.ClientExport(ctx, api.ExportRef{
		Root:         *imp.Root,
		FromLocalCAR: imp.CARPath,
	}, api.FileRef{
		Path:  output,
		IsCAR: false,
	}))

	require.FileExists(s.T(), filepath.Join(output, "file.txt"))

	actualContent, err := os.ReadFile(filepath.Join(output, "file.txt"))
	require.NoError(s.T(), err)

	assert.Equal(s.T(), expectedContent, string(actualContent))
}

func (s *FilecoinPublisherSuite) retrieveUploadedFile(ctx context.Context, contentCidStr string) api.Import {
	contentCid, err := cid.Parse(contentCidStr)
	require.NoError(s.T(), err)

	// Would be nice to be able to do this 'properly'
	imports, err := s.driver.client.ClientListImports(ctx)
	require.NoError(s.T(), err)

	for _, imp := range imports {
		if imp.Root != nil && imp.Root.Equals(contentCid) {
			return imp
		}
	}

	require.Fail(s.T(), "can't find uploaded file", "deal %v not found within %v", contentCid, imports)
	return api.Import{}
}
