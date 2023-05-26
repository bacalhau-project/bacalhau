package repo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// git "github.com/gogs/git-module"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/estuary"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	apicopy "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Response struct {
	CID string
}

type StorageProvider struct {
	LocalDir      string
	EstuaryAPIKey string
	CloneClient   *clone.Clone
	IPFSClient    *apicopy.StorageProvider
}

func NewStorage(cm *system.CleanupManager, IPFSapiclient *apicopy.StorageProvider, EstuaryAPIKey string) (*StorageProvider, error) {
	c, err := clone.NewCloneClient()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(config.GetStoragePath(), "bacalhau-repo")
	if err != nil {
		return nil, err
	}
	cm.RegisterCallback(func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to remove storage folder: %w", err)
		}
		return nil
	})
	storageHandler := &StorageProvider{
		LocalDir:      dir,
		EstuaryAPIKey: EstuaryAPIKey,
		IPFSClient:    IPFSapiclient,
		CloneClient:   c,
	}
	log.Debug().Msgf("Repo download driver created with output dir: %s", dir)
	return storageHandler, nil
}

func (sp *StorageProvider) IsInstalled(context.Context) (bool, error) {
	err := checkGitLFS()
	return err == nil, err
}

func (sp *StorageProvider) HasStorageLocally(context.Context, spec.Storage) (bool, error) {
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(context.Context, spec.Storage) (uint64, error) {
	return 0, nil
}

//nolint:gocyclo
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec spec.Storage) (storage.StorageVolume, error) {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/repo/repo.PrepareStorage")
	defer span.End()

	gitspec, err := git.Decode(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	_, err = clone.IsValidGitRepoURL(gitspec.Repo)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// # create a tmp directory
	outputPath, err := os.MkdirTemp(sp.LocalDir, "*")
	log.Ctx(ctx).Debug().Str("Output Path", outputPath).Msg("created temp folder for repo")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// The `Output` method executes the command and
	// collects the output, returning its value
	cmd := exec.Command("git", "clone", gitspec.Repo, outputPath)
	out, err := cmd.Output()
	log.Ctx(ctx).Debug().Msgf("git clone output is %s", string(out))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("repository", gitspec.Repo).Msg("failed to clone repository")
		return storage.StorageVolume{}, err
	}

	repoPath, err := url.Parse(gitspec.Repo)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	filename := strings.Split(repoPath.Path, ".")[0]
	targetPath := "/inputs" + filename

	CIDSpec, err := sp.Upload(ctx, outputPath)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	ipfsspec, err := spec_ipfs.Decode(CIDSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// Update the KV store
	SHA1HASH, _ := urltoLatestCommitHash(ctx, gitspec.Repo)
	envkey := os.Getenv("ESTUARY_API_KEY")
	if envkey != "" {
		log.Ctx(ctx).Debug().Stringer("CID", ipfsspec.CID).Msg("Pinning CID to estuary")
		err := estuary.PinToIPFSViaEstuary(ctx, envkey, ipfsspec.CID.String())
		if err != nil {
			return storage.StorageVolume{}, err
		}
		log.Ctx(ctx).Debug().Stringer("CID", ipfsspec.CID).Msg("successfully pinned to estuary")
	}

	data := url.Values{}
	data.Set("key", SHA1HASH)
	data.Set("value", ipfsspec.CID.String())

	err = createSHA1CIDPair(ctx, data)
	if err != nil {
		// Although this is an error, it isn't a critical error and we should
		// continue executing after logging the failure.
		log.Ctx(ctx).Error().Err(err).Msg("failed to create SHA1CIDPair")
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
		Target: targetPath,
	}

	return volume, nil
}

func (sp *StorageProvider) Upload(ctx context.Context, localPath string) (spec.Storage, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/repo/apicopy.Upload")
	defer span.End()

	strgspec, err := sp.IPFSClient.Upload(ctx, localPath)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("repo provider failed to upload to IPFS")
		return spec.Storage{}, err
	}

	// NB(frrist): we are only decoding this for the sake logging; remove if this is a performance bottleneck (doubt it will be)
	ipfsspec, err := spec_ipfs.Decode(strgspec)
	if err != nil {
		return spec.Storage{}, err
	}
	log.Ctx(ctx).Debug().Stringer("CID", ipfsspec.CID).Msg("repo provider uploaded to ipfs")

	return strgspec, nil
}

func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	_ spec.Storage,
	volume storage.StorageVolume,
) error {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/repo/repo.CleanupStorage")
	defer span.End()

	pathToCleanup := filepath.Dir(volume.Source)
	log.Ctx(ctx).Debug().Str("Path", pathToCleanup).Msg("Cleaning up")
	return os.RemoveAll(pathToCleanup)
}

func createSHA1CIDPair(ctx context.Context, data url.Values) error {
	//nolint:noctx
	resp, err := http.PostForm("http://kv.bacalhau.org", data)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to write to kv")
		return err
	}
	defer resp.Body.Close()

	log.Ctx(ctx).Debug().Int("status-code", resp.StatusCode).Msg("posting to kv")
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post to kv.bacalhau.org, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to read kv response")
		return err
	}

	log.Ctx(ctx).Debug().Msgf("kv response: %s", body)
	return nil
}

func urltoLatestCommitHash(ctx context.Context, urlStr string) (string, error) {
	cmd := exec.Command("git", "ls-remote", urlStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get latest commit hash")
		return "", err
	}

	x := fmt.Sprintf("%v", string(output)[:40])
	return x, err
}

func checkGitLFS() error {
	_, err := exec.LookPath("git-lfs")
	return err
}
