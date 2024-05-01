//go:build unit || !integration

package storage_test

import (
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
)

// TODO(forrest) [correctness]: understand the intention of these tests and
// strive to remove any specific storage provider implementation from the testing logic.
// It appears these tests aim to validation functionality of ParallelPrepareStorage and ParallelCleanStorage
// they prepare storage, assert files form it are present, then clean up the storage and assert the files were removed
// The proposed solution for testing here is to use mocked storage and assert the parallel storage methods
// make the right calls in the right order. I am dubious on the value of this.

/*
type ParallelStorageSuite struct {
	suite.Suite

	ctx      context.Context
	cm       *system.CleanupManager
	node     *ipfs.Node
	cid      string
	provider provider.Provider[storage.Storage]
}

func TestParallelStorageSuite(t *testing.T) {
	suite.Run(t, new(ParallelStorageSuite))
}

func (s *ParallelStorageSuite) SetupSuite() {
	s.ctx = context.Background()
	s.cm = system.NewCleanupManager()

	// Setup required IPFS node and client
	node, err := ipfs.NewNodeWithConfig(s.ctx, s.cm, types.IpfsConfig{PrivateInternal: true})
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


*/
