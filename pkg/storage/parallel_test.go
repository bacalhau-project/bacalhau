//go:build unit || !integration

package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type ParallelStorageSuite struct {
	suite.Suite

	ctx      context.Context
	cm       *system.CleanupManager
	node     *ipfs.Node
	cid      string
	provider model.Provider[model.StorageSourceType, storage.Storage]
}

func TestParallelStorageSuite(t *testing.T) {
	suite.Run(t, new(ParallelStorageSuite))
}

func (s *ParallelStorageSuite) SetupSuite() {
	s.ctx = context.Background()
	s.cm = system.NewCleanupManager()

	// Setup required IPFS node and client
	node, err := ipfs.NewLocalNode(s.ctx, s.cm, []string{})
	require.NoError(s.T(), err)
	s.node = node
	client := s.node.Client()

	s.cid, err = client.Put(s.ctx, "../../testdata/grep_file.txt")
	require.NoError(s.T(), err)

	s.provider, _ = executor_util.NewStandardStorageProvider(
		s.ctx,
		s.cm,
		executor_util.StandardStorageProviderOptions{
			API: client,
		},
	)
}

func (s *ParallelStorageSuite) TearDownSuite() {
	s.cm.Cleanup(s.ctx)
	_ = s.node.Close(s.ctx)
}

func (s *ParallelStorageSuite) TestIPFSCleanup() {
	volumes, err := storage.ParallelPrepareStorage(s.ctx, s.provider, []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			Name:          "test",
			CID:           s.cid,
			Path:          "/inputs/test.txt",
		},
	}...)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	var files []string
	for _, v := range volumes {
		files = append(files, v.Volume.Source)
	}
	// IPFS cleanup doesn't actually return an error as it deletes a folder
	_ = storage.ParallelCleanStorage(s.ctx, s.provider, volumes)

	// Check that all of the files have gone by statting them and expecting
	// an error for each one
	lo.ForEach(files, func(filepath string, index int) {
		_, err := os.Stat(filepath)
		require.Error(s.T(), err, "file still exists and we expected it to be deleted")
	})
}

func (s *ParallelStorageSuite) TestURLCleanup() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("hello world"))
		s.NoError(err)
	}))
	defer ts.Close()

	volumes, err := storage.ParallelPrepareStorage(s.ctx, s.provider, []model.StorageSpec{
		{
			StorageSource: model.StorageSourceURLDownload,
			Name:          "test",
			URL:           fmt.Sprintf("%s/test.txt", ts.URL),
			Path:          "/inputs/test.txt",
		},
	}...)
	require.NoError(s.T(), err)

	// Make a list of which files we expect to find written to local disk and check they are
	// there.
	var files []string
	for _, v := range volumes {
		files = append(files, v.Volume.Source)
	}

	// URL cleanup doesn't actually return an error as it deletes a folder
	_ = storage.ParallelCleanStorage(s.ctx, s.provider, volumes)

	// Check that all of the files have gone by statting them and expecting
	// an error for each one
	lo.ForEach(files, func(filepath string, index int) {
		_, err := os.Stat(filepath)
		require.Error(s.T(), err, "file still exists and we expected it to be deleted")
	})
}
