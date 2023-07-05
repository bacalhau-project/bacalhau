package parse

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

const defaultStoragePath = "/inputs"

func ParseStorageString(sourceURI, destinationPath string, options map[string]string) (v1beta2.StorageSpec, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return v1beta2.StorageSpec{}, err
	}

	var res v1beta2.StorageSpec
	switch parsedURI.Scheme {
	case "ipfs":
		res = v1beta2.StorageSpec{
			StorageSource: v1beta2.StorageSourceIPFS,
			CID:           parsedURI.Host,
		}
	case "http", "https":
		u, err := urldownload.IsURLSupported(sourceURI)
		if err != nil {
			return v1beta2.StorageSpec{}, err
		}
		res = v1beta2.StorageSpec{
			StorageSource: v1beta2.StorageSourceURLDownload,
			URL:           u.String(),
		}
	case "s3":
		res = v1beta2.StorageSpec{
			StorageSource: v1beta2.StorageSourceS3,
			S3: &v1beta2.S3StorageSpec{
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
				return v1beta2.StorageSpec{}, fmt.Errorf("unknown option %s", key)
			}
		}
	case "file":
		res = v1beta2.StorageSpec{
			StorageSource: v1beta2.StorageSourceLocalDirectory,
			SourcePath:    filepath.Join(parsedURI.Host, parsedURI.Path),
		}
		for key, value := range options {
			switch key {
			case "ro", "read-only", "read_only", "readonly":
				readonly, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return v1beta2.StorageSpec{}, fmt.Errorf("failed to parse read-only option: %s", parseErr)
				}
				res.ReadWrite = !readonly
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return v1beta2.StorageSpec{}, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				res.ReadWrite = readwrite
			default:
				return v1beta2.StorageSpec{}, fmt.Errorf("unknown option %s", key)
			}
		}
	case "git", "gitlfs":
		u, err := clone.IsValidGitRepoURL(sourceURI)
		if err != nil {
			return v1beta2.StorageSpec{}, err
		}
		storageSource := v1beta2.StorageSourceRepoClone
		if parsedURI.Scheme == "gitlfs" {
			storageSource = v1beta2.StorageSourceRepoCloneLFS
		}
		res = v1beta2.StorageSpec{
			StorageSource: storageSource,
			Repo:          u.String(),
		}
	default:
		return v1beta2.StorageSpec{}, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}
	res.Name = sourceURI
	res.Path = destinationPath
	if res.Path == "" {
		res.Path = defaultStoragePath
	}
	return res, nil
}

func ParsePublisherString(destinationURI string, options map[string]interface{}) (v1beta2.PublisherSpec, error) {
	destinationURI = strings.Trim(destinationURI, " '\"")
	parsedURI, err := url.Parse(destinationURI)
	if err != nil {
		return v1beta2.PublisherSpec{}, err
	}

	// handle scenarios where the destinationURI is just the scheme/publisher type, e.g. ipfs
	if parsedURI.Scheme == "" {
		parsedURI.Scheme = parsedURI.Path
	}

	var res v1beta2.PublisherSpec
	switch parsedURI.Scheme {
	case "ipfs":
		res = v1beta2.PublisherSpec{
			Type: v1beta2.PublisherIpfs,
		}
	case "estuary":
		res = v1beta2.PublisherSpec{
			Type: v1beta2.PublisherEstuary,
		}
	case "s3":
		if _, ok := options["bucket"]; !ok {
			options["bucket"] = parsedURI.Host
		}
		if _, ok := options["key"]; !ok {
			options["key"] = strings.TrimLeft(parsedURI.Path, "/")
		}
		res = v1beta2.PublisherSpec{
			Type:   v1beta2.PublisherS3,
			Params: options,
		}
	default:
		return v1beta2.PublisherSpec{}, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}
	return res, nil
}
