package legacyjob

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/clone"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

const defaultStoragePath = "/inputs"

const (
	s3Prefix   = "s3"
	ipfsPrefix = "ipfs"
)

//nolint:gocyclo,funlen
func ParseStorageString(sourceURI, destinationPath string, options map[string]string) (model.StorageSpec, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return model.StorageSpec{}, err
	}

	var res model.StorageSpec
	switch parsedURI.Scheme {
	case ipfsPrefix:
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
	case "inline":
		inline := inline.NewStorage()
		specConfig, err := inline.Upload(context.Background(), sourceURI)
		if err != nil {
			return model.StorageSpec{}, err
		}

		fmt.Println("-------------------------")
		fmt.Printf("%+v\n", specConfig.Params)
		fmt.Println("-------------------------")

		res = model.StorageSpec{
			StorageSource: model.StorageSourceInline,
			URL:           specConfig.Params["URL"].(string),
			Path:          destinationPath,
		}
	case s3Prefix:
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
	case "file":
		res = model.StorageSpec{
			StorageSource: model.StorageSourceLocalDirectory,
			SourcePath:    filepath.Join(parsedURI.Host, parsedURI.Path),
		}
		for key, value := range options {
			switch key {
			case "ro", "read-only", "read_only", "readonly":
				readonly, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return model.StorageSpec{}, fmt.Errorf("failed to parse read-only option: %s", parseErr)
				}
				res.ReadWrite = !readonly
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return model.StorageSpec{}, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				res.ReadWrite = readwrite
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

// RawPublisherStringToPublisherSpec is a legacy function for converting a raw string
// into a model.PublisherSpec.
func RawPublisherStringToPublisherSpec(publisher string) (model.PublisherSpec, error) {
	spec := model.PublisherSpec{}

	publisher = strings.Trim(publisher, " '\"")
	csvReader := csv.NewReader(strings.NewReader(publisher))
	fields, err := csvReader.Read()
	if err != nil {
		return spec, err
	}

	var destinationURI string
	options := make(map[string]interface{})

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of just publisher type
			if i == 0 {
				destinationURI = field
				continue
			} else {
				return spec, fmt.Errorf("invalid publisher option: %s. Must be a key=value pair", field)
			}
		}

		key = strings.ToLower(key)
		switch key {
		case "target", "dst", "destination":
			destinationURI = val
		case "opt", "option":
			k, v, _ := strings.Cut(val, "=")
			if k != "" {
				options[k] = v
			}
		default:
			return spec, fmt.Errorf("invalid publisher option: %s", field)
		}
	}

	return PublisherStringToPublisherSpec(destinationURI, options)
}

func PublisherStringToPublisherSpec(destinationURI string, options map[string]interface{}) (model.PublisherSpec, error) {
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
	case ipfsPrefix:
		res = model.PublisherSpec{
			Type: model.PublisherIpfs,
		}
	case s3Prefix:
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
	case "local":
		res = model.PublisherSpec{
			Type: model.PublisherLocal,
		}
	default:
		return model.PublisherSpec{}, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}
