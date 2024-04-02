package clone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rs/zerolog/log"

	"net/http"
	"net/url"
)

const baseURL = "http://kv.bacalhau.org/"

type Clone struct {
	URL string
}

type Response struct {
	CID string
}

func NewCloneClient() (*Clone, error) {
	return &Clone{
		URL: "",
	}, nil
}

func RepoExistsOnIPFSGivenURL(ctx context.Context, urlStr string) (string, error) {
	output, err := getLatestCommitHash(urlStr)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get latest commit hash")
		return "", err
	}

	url := baseURL + output
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create new http request to kv")
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to make http request to kv")
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to ready body from kv response")
		return "", err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshall kv response")
		return "", err
	}

	return response.CID, nil
}

func RemoveFromModelStorageSpec(inputs []model.StorageSpec, url string) []model.StorageSpec {
	newArr := []model.StorageSpec{}
	for _, s := range inputs {
		if s.StorageSource == model.StorageSourceRepoClone {
			if s.Repo != url {
				newArr = append(newArr, model.StorageSpec{
					StorageSource: model.StorageSourceRepoClone,
					Repo:          url,
					Path:          "/inputs",
				})
			}
		}
	}
	return newArr
}

func IsValidGitRepoURL(urlStr string) (*url.URL, error) {
	// Check if the URL string is empty
	if urlStr == "" {
		return nil, fmt.Errorf("URL is empty")
	}
	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "gitlfs" || u.Scheme == "git" {
		u.Scheme = "https"
	}
	// Check if the URL is a Git repository URL
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "ssh" {
		return nil, fmt.Errorf("URL must use HTTPS, HTTP, or SSH scheme")
	}
	if !strings.HasSuffix(u.Path, ".git") {
		return nil, fmt.Errorf("URL must use .git file extension")
	}
	return u, nil
}

func getLatestCommitHash(URL string) (string, error) {
	// Create a memory storage
	memStorage := memory.NewStorage()

	// Clone the remote repository into the memory storage
	repo, err := git.Clone(memStorage, nil, &git.CloneOptions{
		URL:          URL,
		Depth:        1,
		SingleBranch: true,
		NoCheckout:   true,
	})
	if err != nil {
		return "", err
	}

	// Get the reference to the head of the repository
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	// Get the hash of the latest commit
	commitHash := headRef.Hash()

	return commitHash.String(), nil
}
