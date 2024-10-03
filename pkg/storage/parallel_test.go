//go:build integration || !unit

package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type ParallelStorageSuite struct {
	suite.Suite
	cfg      types.Bacalhau
	provider provider.Provider[storage.Storage]
}

func TestParallelStorageSuite(t *testing.T) {
	suite.Run(t, new(ParallelStorageSuite))
}

func (s *ParallelStorageSuite) SetupSuite() {
	var err error
	s.cfg, err = config.NewTestConfig()
	s.Require().NoError(err)

	s.provider, err = executor_util.NewStandardStorageProvider(s.cfg)
	s.Require().NoError(err)
}

func (s *ParallelStorageSuite) TestIPFSCleanup() {
	endpoint := testutils.MustHaveIPFSEndpoint(s.T())

	ctx := context.Background()
	client, err := ipfs.NewClient(ctx, endpoint)
	require.NoError(s.T(), err)

	cid, err := client.Put(ctx, "../../testdata/grep_file.txt")
	require.NoError(s.T(), err)

	artifact := &models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceIPFS,
			Params: map[string]interface{}{
				"CID": cid,
			},
		},
		Target: "/inputs/test.txt",
	}
	volumes, err := storage.ParallelPrepareStorage(ctx, s.provider, s.T().TempDir(), artifact)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	for _, v := range volumes {
		s.Require().FileExists(v.Volume.Source)
	}

	// Cleanup the directory and make sure there are no longer any assets left
	err = storage.ParallelCleanStorage(ctx, s.provider, volumes)
	s.Require().NoError(err)

	for _, v := range volumes {
		s.Require().NoFileExists(v.Volume.Source)
	}
}

func (s *ParallelStorageSuite) TestURLCleanup() {
	ctx := context.Background()
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

	volumes, err := storage.ParallelPrepareStorage(ctx, s.provider, s.T().TempDir(), artifact)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	for _, v := range volumes {
		s.Require().FileExists(v.Volume.Source)
	}

	err = storage.ParallelCleanStorage(ctx, s.provider, volumes)
	s.Require().NoError(err)

	for _, v := range volumes {
		s.Require().NoFileExists(v.Volume.Source)
	}
}
