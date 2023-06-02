//go:build !unit || integration

// TODO(forrest): this test requires network access and creating an IPFS node there for it should be an integration test.

package repo

import (
	"context"
	"fmt"
	// "net/http"
	// "net/http/httptest"
	"os"
	"path/filepath"
	// "regexp"
	"testing"

	"github.com/go-git/go-git/v5"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	spec_git "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	apicopy "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type StorageSuite struct {
	suite.Suite
	RootCmd *cobra.Command
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageSuite))
}

// Before each test
func (s *StorageSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	system.InitConfigForTesting(s.T())
}

func getIpfsStorage() (*apicopy.StorageProvider, error) {
	ctx := context.Background()
	cm := system.NewCleanupManager()

	node, err := ipfs.NewLocalNode(ctx, cm, []string{})
	if err != nil {
		// panic(err)
		return nil, err

	}
	// // require.NoError(t, err)

	// apiAddresses, err := node.APIAddresses()
	if err != nil {
		// panic(err)
		return nil, err

	}
	cl := ipfs.NewClient(node.Client().API)

	storage, err := apicopy.NewStorage(cm, cl)
	if err != nil {
		// panic(err)
		return nil, err
	}

	return storage, nil
}

func (s *StorageSuite) TestNewStorageProvider() {
	cm := system.NewCleanupManager()
	storage, err := getIpfsStorage()
	if err != nil {
		panic(err)
	}
	sp, err := NewStorage(cm, storage, "")
	require.NoError(s.T(), err, "failed to create storage provider")

	// is dir writable?
	fmt.Println(sp.LocalDir)
	f, err := os.Create(filepath.Join(sp.LocalDir, "data.txt"))
	require.NoError(s.T(), err, "failed to create file")

	_, err = f.WriteString("test\n")
	require.NoError(s.T(), err, "failed to write to file")

	require.NoError(s.T(), f.Close())
	// if sp.IPFSClient == nil {
	// 	require.Fail(s.T(), "IPFSClient is nil")
	// }
}

func (s *StorageSuite) TestHasStorageLocally() {
	cm := system.NewCleanupManager()
	ctx := context.Background()
	storage, err := getIpfsStorage()
	if err != nil {
		panic(err)
	}
	sp, err := NewStorage(cm, storage, "")
	require.NoError(s.T(), err, "failed to create storage provider")

	gitSpec, err := (&spec_git.GitStorageSpec{Repo: "foo"}).AsSpec("TODO", "foo")
	require.NoError(s.T(), err)
	// files are not cached thus shall never return true
	locally, err := sp.HasStorageLocally(ctx, gitSpec)
	require.NoError(s.T(), err, "failed to check if storage is locally available")

	if locally != false {
		require.Fail(s.T(), "storage should not be locally available")
	}
}

func (s *StorageSuite) TestCloneRepo() {
	// This test will fail when offline - we should build a checker to see if someone
	// is connected to the internet and skip this test if they are not.
	// This test will also fail if the URL is not reachable.
	// Using -test.short flag for now
	if testing.Short() {
		s.T().Skip("Skipping test that requires internet connection")
	}
	type repostruct struct {
		Site     string
		URL      string
		repoName string
	}
	// Rewrite this test replacing it with the clone part
	filetypeCases := []repostruct{
		{Site: "github", URL: "https://github.com/bacalhau-project/bacalhau.git",
			repoName: "bacalhau-project/bacalhau",
		}}

	for _, ftc := range filetypeCases {
		name := fmt.Sprintf("%s-%s", ftc.Site, ftc.URL)

		hash, err := func() (string, error) {
			cm := system.NewCleanupManager()
			ctx := context.Background()
			storage, err := getIpfsStorage()
			if err != nil {
				panic(err)
			}
			sp, err := NewStorage(cm, storage, "")
			if err != nil {
				return "", fmt.Errorf("%s: failed to create storage provider", name)
			}

			gitSpec, err := (&spec_git.GitStorageSpec{Repo: ftc.URL}).
				AsSpec("TODO", "/inputs/"+ftc.repoName)
			if err != nil {
				return "", err
			}

			volume, err := sp.PrepareStorage(ctx, gitSpec)
			if err != nil {
				return "", fmt.Errorf("%s: failed to prepare storage: %+v", name, err)
			}

			r, err := git.PlainOpen(volume.Source)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			ref, err := r.Head()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			commit, err := r.CommitObject(ref.Hash())
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			headhash := commit.Hash.String()
			return headhash, nil
		}()
		if err != nil {
			fmt.Print(err)
		}
		urlhash, _ := urltoLatestCommitHash(context.Background(), ftc.URL)
		if urlhash != "" {
			require.Equal(s.T(), urlhash, hash, "%s: content of file does not match", name)
		}
		fmt.Printf("HASH: %s", hash)

	}
}
