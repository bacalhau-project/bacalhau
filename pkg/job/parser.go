package job

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/gitlfs"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	spec_local "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	spec_url "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

const defaultStoragePath = "/inputs"

type specable interface {
	AsSpec(name, mount string) (spec.Storage, error)
}

func ParseStorageString(sourceURI, destinationPath string, options map[string]string) (spec.Storage, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return spec.Storage{}, err
	}

	var res specable
	switch parsedURI.Scheme {
	case "ipfs":
		c, err := cid.Decode(parsedURI.Host)
		if err != nil {
			return spec.Storage{}, nil
		}
		res = &spec_ipfs.IPFSStorageSpec{CID: c}

	case "http", "https":
		u, err := urldownload.IsURLSupported(sourceURI)
		if err != nil {
			return spec.Storage{}, err
		}
		res = &spec_url.URLStorageSpec{URL: u.String()}

	case "s3":
		s3spec := &spec_s3.S3StorageSpec{
			Bucket: parsedURI.Host,
			Key:    strings.TrimLeft(parsedURI.Path, "/"),
		}
		for key, value := range options {
			switch key {
			case "endpoint":
				s3spec.Endpoint = value
			case "region":
				s3spec.Region = value
			case "versionID", "version-id", "version_id":
				s3spec.VersionID = value
			case "checksum-256", "checksum256", "checksum_256":
				s3spec.ChecksumSHA256 = value
			default:
				return spec.Storage{}, fmt.Errorf("unknown option %s", key)
			}
		}
		res = s3spec

	case "file":
		var rw bool
		for key, value := range options {
			switch key {
			case "ro", "read-only", "read_only", "readonly":
				readonly, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return spec.Storage{}, fmt.Errorf("failed to parse read-only option: %s", parseErr)
				}
				rw = !readonly
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return spec.Storage{}, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				rw = readwrite
			default:
				return spec.Storage{}, fmt.Errorf("unknown option %s", key)
			}
		}
		res = &spec_local.LocalStorageSpec{Source: filepath.Join(parsedURI.Host, parsedURI.Path), ReadWrite: rw}

	case "git", "gitlfs":
		u, err := clone.IsValidGitRepoURL(sourceURI)
		if err != nil {
			return spec.Storage{}, err
		}
		if parsedURI.Scheme == "gitlfs" {
			res = &gitlfs.GitLFSStorageSpec{Repo: u.String()}
		} else {
			res = &git.GitStorageSpec{Repo: u.String()}
		}

	default:
		return spec.Storage{}, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}
	if destinationPath == "" {
		return res.AsSpec(sourceURI, defaultStoragePath)
	}
	return res.AsSpec(sourceURI, destinationPath)
}

func ParsePublisherString(destinationURI string, options map[string]interface{}) (model.PublisherSpec, error) {
	destinationURI = strings.Trim(destinationURI, " '\"")
	parsedURI, err := url.Parse(destinationURI)
	if err != nil {
		return model.PublisherSpec{}, err
	}

	// handle scenarios where the destinationURI is just the scheme/publisher type, e.g. ipfs
	if parsedURI.Scheme == "" {
		parsedURI.Scheme = parsedURI.Path
	}

	var res model.PublisherSpec
	switch parsedURI.Scheme {
	case "ipfs":
		res = model.PublisherSpec{
			Type: model.PublisherIpfs,
		}
	case "lotus":
		res = model.PublisherSpec{
			Type: model.PublisherFilecoin,
		}
	case "estuary":
		res = model.PublisherSpec{
			Type: model.PublisherEstuary,
		}
	case "s3":
		if _, ok := options["bucket"]; !ok {
			options["bucket"] = parsedURI.Host
		}
		if _, ok := options["key"]; !ok {
			options["key"] = strings.TrimLeft(parsedURI.Path, "/")
		}
		res = model.PublisherSpec{
			Type:   model.PublisherS3,
			Params: options,
		}
	default:
		return model.PublisherSpec{}, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}
	return res, nil
}
