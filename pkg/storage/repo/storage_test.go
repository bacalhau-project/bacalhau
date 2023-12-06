//go:build unit || !integration

package repo

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	apicopy "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type StorageSuite struct {
	suite.Suite
	RootCmd *cobra.Command

	ctx context.Context
	cm  *system.CleanupManager

	node   *ipfs.Node
	client ipfs.Client

	copyProvider *apicopy.StorageProvider
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

func (s *StorageSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	setup.SetupBacalhauRepoForTesting(s.T())

	s.ctx = context.Background()
	s.cm = system.NewCleanupManager()

	node, _ := ipfs.NewNodeWithConfig(s.ctx, s.cm, types.IpfsConfig{PrivateInternal: true})
	s.node = node

	s.client = ipfs.NewClient(s.node.Client().API)
	s.copyProvider, _ = apicopy.NewStorage(s.client)
}

func (s *StorageSuite) TearDownSuite() {
	s.node.Close(s.ctx)
}

func (s *StorageSuite) TestHasStorageLocally() {
	ctx := context.Background()

	sp, err := NewStorage(s.copyProvider)
	s.Require().NoError(err, "failed to create storage provider")

	spec := models.InputSource{
		Source: &models.SpecConfig{
			Type: models.StorageSourceRepoClone,
			Params: Source{
				Repo: "foo",
			}.ToMap(),
		},
		Target: "bar",
	}

	// files are not cached thus shall never return true
	locally, err := sp.HasStorageLocally(ctx, spec)
	s.Require().NoError(err, "failed to check if storage is locally available")

	if locally != false {
		s.Fail("storage should not be locally available")
	}
}

func (s *StorageSuite) TestCloneRepo() {
	// This test will fail when offline - we should build a checker to see if someone
	// is connected to the internet and skip this test if they are not.
	// This test will also fail if the URL is not reachable.
	// Using -test.short flag for now
	// s.T().Skip("Skipping test that requires internet connection")

	type repostruct struct {
		name     string
		url      string
		repoName string
	}

	tmpDirectory := s.T().TempDir()
	projectDirectory := "test/project.git"

	projectDirectoryActual := path.Join(tmpDirectory, projectDirectory)
	err := os.MkdirAll(projectDirectoryActual, 0755)
	s.Require().NoError(err)

	exampleFile := filepath.Join(projectDirectoryActual, "example.txt")
	err = os.WriteFile(exampleFile, []byte("hello"), 0755)
	s.Require().NoError(err)

	// Create and initialise the local git server
	gs, err := NewGitServer(tmpDirectory, projectDirectory)
	s.Require().NoError(err)

	err = gs.Init("example.txt")
	s.Require().NoError(err)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c, err := gs.Serve(ctx)
	s.Require().NoError(err)

	defer func() {
		fmt.Println("Calling cancel")
		c.Process.Kill()
	}()

	// Rewrite this test replacing it with the clone part
	filetypeCases := []repostruct{
		{
			name:     "simple clone",
			url:      "git://127.0.0.1:9418/test/project.git",
			repoName: "test/project",
		}}

	for _, ftc := range filetypeCases {
		s.Run(ftc.name, func() {
			sp, err := NewStorage(s.copyProvider)
			s.Require().NoError(err)

			spec := models.InputSource{
				Source: &models.SpecConfig{
					Type: models.StorageSourceRepoClone,
					Params: Source{
						Repo: ftc.url,
					}.ToMap(),
				},
				Target: "/inputs/" + ftc.repoName,
			}

			tempRunFolder := s.T().TempDir()
			volume, err := sp.PrepareStorage(ctx, tempRunFolder, spec)
			s.Require().NoError(err)

			r, err := git.PlainOpen(volume.Source)
			s.Require().NoError(err, "failed to call git.PlainOpen on %s", volume.Source)

			ref, err := r.Head()
			s.Require().NoError(err)

			commit, err := r.CommitObject(ref.Hash())
			s.Require().NoError(err)

			headhash := commit.Hash.String()

			urlhash, _ := urltoLatestCommitHash(context.Background(), ftc.url)
			if urlhash != "" {
				s.Require().Equal(urlhash, headhash, "%s: content of file does not match", ftc.name)
			}

		})
	}
}
