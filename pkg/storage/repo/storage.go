package repo

import (
	"context"
	"fmt"
	"io"
	"os"

	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	// git "github.com/gogs/git-module"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	apicopy "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Response struct {
	CID string
}

type StorageProvider struct {
	CloneClient *clone.Clone
	IPFSClient  *apicopy.StorageProvider
}

func NewStorage(IPFSapiclient *apicopy.StorageProvider) (*StorageProvider, error) {
	c, err := clone.NewCloneClient()
	if err != nil {
		return nil, err
	}
	storageHandler := &StorageProvider{
		IPFSClient:  IPFSapiclient,
		CloneClient: c,
	}
	log.Debug().Msgf("Repo download driver created")
	return storageHandler, nil
}

func (sp *StorageProvider) IsInstalled(context.Context) (bool, error) {
	err := checkGitLFS()
	return err == nil, err
}

func (sp *StorageProvider) HasStorageLocally(context.Context, models.InputSource) (bool, error) {
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(context.Context, models.InputSource) (uint64, error) {
	return 0, nil
}

//nolint:gocyclo
func (sp *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageDirectory string,
	storageSpec models.InputSource) (storage.StorageVolume, error) {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/repo/repo.PrepareStorage")
	defer span.End()

	source, err := DecodeSpec(storageSpec.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	repoURL := source.Repo
	_, err = clone.IsValidGitRepoURL(repoURL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// # create a tmp directory inside the provided directory
	outputPath, err := os.MkdirTemp(storageDirectory, "*")
	log.Ctx(ctx).Debug().Str("Output ResultPath", outputPath).Msg("created temp folder for repo")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// The `Output` method executes the command and
	// collects the output, returning its value
	cmd := exec.Command("git", "clone", repoURL, outputPath)
	out, err := cmd.Output()
	log.Ctx(ctx).Debug().Msgf("git clone output is %s", string(out))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("repository", repoURL).Msg("failed to clone repository")
		return storage.StorageVolume{}, err
	}

	filepath, err := url.Parse(repoURL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	filename := strings.Split(filepath.Path, ".")[0]
	targetPath := "/inputs" + filename

	outputConfig, err := sp.Upload(ctx, outputPath)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	cid := outputConfig.Params["CID"].(string)
	// Update the KV store
	SHA1HASH, _ := urltoLatestCommitHash(ctx, repoURL)

	data := url.Values{}
	data.Set("key", SHA1HASH)
	data.Set("value", cid)

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

func (sp *StorageProvider) Upload(ctx context.Context, localPath string) (models.SpecConfig, error) {
	return sp.IPFSClient.Upload(ctx, localPath)
}

func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	_ models.InputSource,
	volume storage.StorageVolume,
) error {
	return os.Remove(volume.Source)
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
