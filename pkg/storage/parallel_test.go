//go:build unit || !integration

package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"

	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type ParallelStorageSuite struct {
	suite.Suite

	ctx         context.Context
	cm          *system.CleanupManager
	ipfsEnabled bool
	cid         string
	provider    provider.Provider[storage.Storage]
}

func TestParallelStorageSuite(t *testing.T) {
	suite.Run(t, new(ParallelStorageSuite))
}

func (s *ParallelStorageSuite) SetupSuite() {
	s.ctx = context.Background()
	_, cfg := setup.SetupBacalhauRepoForTesting(s.T())

	s.ipfsEnabled = testutils.IsIPFSEnabled(cfg.Node.IPFS.Connect)

	var err error
	s.provider, err = executor_util.NewStandardStorageProvider(
		time.Duration(configenv.Testing.Node.VolumeSizeRequestTimeout),
		time.Duration(configenv.Testing.Node.DownloadURLRequestTimeout),
		configenv.Testing.Node.DownloadURLRequestRetries,
		executor_util.StandardStorageProviderOptions{IPFSConnect: cfg.Node.IPFS.Connect},
	)
	s.Require().NoError(err)
}

func (s *ParallelStorageSuite) TestIPFSCleanup() {
	if !s.ipfsEnabled {
		s.T().Skip("IPFS connect not configured")
	}

	artifact := &models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceIPFS,
			Params: map[string]interface{}{
				"CID": s.cid,
			},
		},
		Target: "/inputs/test.txt",
	}
	volumes, err := storage.ParallelPrepareStorage(s.ctx, s.provider, s.T().TempDir(), artifact)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	for _, v := range volumes {
		s.Require().FileExists(v.Volume.Source)
	}

	// Cleanup the directory and make sure there are no longer any assets left
	err = storage.ParallelCleanStorage(s.ctx, s.provider, volumes)
	s.Require().NoError(err)

	for _, v := range volumes {
		s.Require().NoFileExists(v.Volume.Source)
	}
}

func (s *ParallelStorageSuite) TestURLCleanup() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("hello world"))
		s.NoError(err)
	}))
	defer ts.Close()

	artifact := &models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceURL,
			Params: map[string]interface{}{
				"URL": fmt.Sprintf("%s/test.txt", ts.URL),
			},
		},
		Target: "/inputs/test.txt",
	}

	volumes, err := storage.ParallelPrepareStorage(s.ctx, s.provider, s.T().TempDir(), artifact)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	for _, v := range volumes {
		s.Require().FileExists(v.Volume.Source)
	}

	err = storage.ParallelCleanStorage(s.ctx, s.provider, volumes)
	s.Require().NoError(err)

	for _, v := range volumes {
		s.Require().NoFileExists(v.Volume.Source)
	}
}
