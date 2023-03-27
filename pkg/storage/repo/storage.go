package repo

import (
	"context"
	"fmt"
	"io"
	log1 "log"
	"os"
	"path/filepath"

	"net/http"
	"net/url"
	"os/exec"
	"strings"

	// git "github.com/gogs/git-module"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

func (sp *StorageProvider) HasStorageLocally(context.Context, model.StorageSpec) (bool, error) {
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(context.Context, model.StorageSpec) (uint64, error) {
	return 0, nil
}

//nolint:gocyclo
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/repo/repo.PrepareStorage")
	defer span.End()
	repoURL := storageSpec.Repo
	var err error
	_, err = clone.IsValidGitRepoURL(repoURL)
	// fmt.Printf("%+v", storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	// # create a tmp directory
	outputPath, err := os.MkdirTemp(sp.LocalDir, "*")
	log.Debug().Str("Output Path", outputPath).Send()
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// err = git.Clone(repoURL, outputPath)
	// if err != nil {
	// 	fmt.Println("Failed to clone repository:", err)
	cmd := exec.Command("git", "clone", repoURL, outputPath)
	// The `Output` method executes the command and
	// collects the output, returning its value
	out, err1 := cmd.Output()
	if err1 != nil {
		// if there was any error, print it here
		fmt.Println("could not run command: ", err)
	}
	// otherwise, print the output from running the command
	fmt.Println("Output: ", string(out))
	if err != nil {
		panic(err)
	}

	if err != nil {
		return storage.StorageVolume{}, err
	}
	// }
	filepath, err2 := url.Parse(repoURL)
	if err2 != nil {
		return storage.StorageVolume{}, err
	}
	filename := strings.Split(filepath.Path, ".")[0]
	targetPath := "/inputs" + filename

	CIDSpec, err := sp.Upload(ctx, outputPath)
	// If estuary key exists then upload it to estuary
	if err != nil {
		return storage.StorageVolume{}, err
	}
	CID := CIDSpec.CID
	// Update the KV store
	SHA1HASH, _ := UrltoLatestCommitHash(repoURL)
	envkey := os.Getenv("ESTUARY_API_KEY")

	if envkey != "" {
		log1.Println("Pinning to Estuary...")
		//nolint:govet,noctx
		err := estuary.PinToIPFSViaEstuary(ctx, envkey, CID)
		if err != nil {
			return storage.StorageVolume{}, err
		}
		log.Print("Successfully Pinned to Estuary...")
	}
	data := url.Values{}
	data.Set("key", SHA1HASH)
	data.Set("value", CID)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	err = CreateSHA1CIDPair(data)
	if err != nil {
		fmt.Printf("Failed to create SHA1CIDPair: %v", err)
	}
	// return the volume
	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
		Target: targetPath,
	}

	return volume, nil
}

func (sp *StorageProvider) Upload(ctx context.Context, localPath string) (model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/repo/apicopy.Upload")
	defer span.End()

	cid, err := sp.IPFSClient.Upload(ctx, localPath)
	fmt.Print(cid)
	if err != nil {
		fmt.Print(err)
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           cid.CID,
	}, nil
}

func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	_ model.StorageSpec,
	volume storage.StorageVolume,
) error {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/repo/repo.CleanupStorage")
	defer span.End()

	pathToCleanup := filepath.Dir(volume.Source)
	log.Ctx(ctx).Debug().Str("Path", pathToCleanup).Msg("Cleaning up")
	return os.RemoveAll(pathToCleanup)
}

func (sp *StorageProvider) Explode(_ context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{
		{
			Name:          spec.Name,
			StorageSource: model.StorageSourceRepoClone,
			Path:          spec.Path,
			Repo:          spec.Repo,
		},
	}, nil
}

func CreateSHA1CIDPair(data url.Values) error {
	//nolint:noctx
	resp, err := http.PostForm("http://kv.bacalhau.org", data)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log1.Println("Request successful")
	} else {
		log1.Println("Request failed")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
		return err
	}
	fmt.Println(string(body))
	return nil
}

func UrltoLatestCommitHash(urlStr string) (string, error) {
	cmd := exec.Command("git", "ls-remote", urlStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	x := fmt.Sprintf("%v", string(output)[:40])
	return x, err
}

func RemoveFromSlice(arr []string, item string) []string {
	newArr := []string{}
	for _, s := range arr {
		if s != item {
			newArr = append(newArr, s)
		}
	}
	return newArr
}

func checkGitLFS() error {
	_, err := exec.LookPath("git-lfs")
	if err != nil {
		return fmt.Errorf("git-lfs is not installed. Please install it first")
	}
	return nil
}
