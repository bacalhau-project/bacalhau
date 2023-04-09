package job

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

const defaultStoragePath = "/inputs"

func ParseStorageString(sourceURI, destinationPath string, options map[string]string) (model.StorageSpec, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return model.StorageSpec{}, err
	}

	var res model.StorageSpec
	switch parsedURI.Scheme {
	case "ipfs":
		res = model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           parsedURI.Host,
		}
	case "http", "https":
		u, err := urldownload.IsURLSupported(sourceURI)
		if err != nil {
			return model.StorageSpec{}, err
		}
		res = model.StorageSpec{
			StorageSource: model.StorageSourceURLDownload,
			URL:           u.String(),
		}
	case "s3":
		res = model.StorageSpec{
			StorageSource: model.StorageSourceS3,
			S3: &model.S3StorageSpec{
				Bucket: parsedURI.Host,
				Key:    strings.TrimLeft(parsedURI.Path, "/"),
			},
		}
		for key, value := range options {
			switch key {
			case "endpoint":
				res.S3.Endpoint = value
			case "region":
				res.S3.Region = value
			case "versionID", "version-id", "version_id":
				res.S3.VersionID = value
			case "checksum-256", "checksum256", "checksum_256":
				res.S3.ChecksumSHA256 = value
			default:
				return model.StorageSpec{}, fmt.Errorf("unknown option %s", key)
			}
		}
	case "git", "gitlfs":
		u, err := clone.IsValidGitRepoURL(sourceURI)
		if err != nil {
			return model.StorageSpec{}, err
		}
		storageSource := model.StorageSourceRepoClone
		if parsedURI.Scheme == "gitlfs" {
			storageSource = model.StorageSourceRepoCloneLFS
		}
		res = model.StorageSpec{
			StorageSource: storageSource,
			Repo:          u.String(),
		}
	default:
		return model.StorageSpec{}, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}
	res.Name = sourceURI
	res.Path = destinationPath
	if res.Path == "" {
		res.Path = defaultStoragePath
	}
	return res, nil
}
