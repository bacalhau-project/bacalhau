package models

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

//nolint:gocyclo
func StorageStringToSpecConfig(sourceURI, destinationPath, alias string, options map[string]string) (*InputSource, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return nil, err
	}

	var sc *SpecConfig
	switch parsedURI.Scheme {
	case ipfsPrefix:
		sc = NewSpecConfig(StorageSourceIPFS)
		sc.WithParam("cid", parsedURI.Host)
	case "http", "https":
		// TODO(forrest) [refactor]: decide if this is where we want validation
		u, err := isURLSupported(sourceURI)
		if err != nil {
			return nil, err
		}
		sc = NewSpecConfig(StorageSourceURL)
		sc.WithParam("url", u.String())
	case s3Prefix:
		sc = NewSpecConfig(StorageSourceS3)
		sc.WithParam("bucket", parsedURI.Host)
		sc.WithParam("key", strings.TrimLeft(parsedURI.Path, "/"))
		for key, value := range options {
			switch key {
			case "endpoint":
				sc.WithParam("endpoint", value)
			case "region":
				sc.WithParam("region", value)
			case "versionID", "version-id", "version_id":
				sc.WithParam("versionid", value)
			case "checksum-256", "checksum256", "checksum_256":
				sc.WithParam("checksum256", value)
			default:
				return nil, fmt.Errorf("unknown option %q for storage %s", key, parsedURI.Scheme)
			}
		}
	case "file":
		sc = NewSpecConfig(StorageSourceLocalDirectory)
		sc.WithParam("sourcepath", filepath.Join(parsedURI.Host, parsedURI.Path))
		for key, value := range options {
			switch key {
			// TODO(forrest) [correctness]: we need some type of validation that its not ro and rw
			case "ro", "read-only", "read_only", "readonly":
				readonly, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse read-only option: %s", parseErr)
				}
				sc.WithParam("readonly", readonly)
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				sc.WithParam("readwrite", readwrite)
			default:
				return nil, fmt.Errorf("unknown option %s", key)
			}
		}
	case "git", "gitlfs":
		return nil, fmt.Errorf("unsupported type: %s", parsedURI.Scheme)
	default:
		return nil, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}

	return &InputSource{
		Source: sc,
		Alias:  alias,
		Target: destinationPath,
	}, nil
}

func isURLSupported(rawURL string) (*url.URL, error) {
	rawURL = strings.Trim(rawURL, " '\"")
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err)
	}
	if (u.Scheme != "http") && (u.Scheme != "https") {
		return nil, fmt.Errorf("URLs must begin with 'http' or 'https'. The submitted one began with %s", u.Scheme)
	}

	basePath := path.Base(u.Path)

	// Need to check for both because a bare host
	// Like http://localhost/ gets converted to "." by path.Base
	if basePath == "" || u.Path == "" {
		return nil, fmt.Errorf("URL must end with a file name")
	}

	return u, nil
}
