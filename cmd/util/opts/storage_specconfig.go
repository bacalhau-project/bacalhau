package opts

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	storage_ipfs "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	storage_local "github.com/bacalhau-project/bacalhau/pkg/storage/local"
	storage_s3 "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	storage_url "github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &StorageSpecConfigOpt{}

type StorageSpecConfigOpt struct {
	values []*models.InputSource
}

// AddValue adds a new storage source to the list of sources
func (o *StorageSpecConfigOpt) AddValue(value *models.InputSource) {
	o.values = append(o.values, value)
}

// ParseStorageSpec parses a storage spec string in the format source:destination or with explicit key=value pairs
// and returns an InputSource. This is extracted from StorageSpecConfigOpt.Set for reuse.
func ParseStorageSpec(value string, defaultDestination string) (*models.InputSource, error) {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	var sourceURI string
	destination := defaultDestination
	options := make(map[string]string)

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of source:destination
			if i == 0 {
				parsedURI, err := url.Parse(field)
				if err != nil {
					return nil, err
				}
				// find the last colon, excluding the schema part
				schema := parsedURI.Scheme
				trimmedURI := strings.TrimPrefix(field, schema+"://")
				index := strings.LastIndex(trimmedURI, ":")
				if index == -1 {
					sourceURI = field
				} else {
					sourceURI = schema + "://" + trimmedURI[:index]
					destination = trimmedURI[index+1:]
				}
				continue
			} else {
				return nil, fmt.Errorf("invalid storage option: %s. Must be a key=value pair", field)
			}
		}

		key = strings.ToLower(key)
		switch key {
		case "source", "src":
			sourceURI = val
		case "target", "dst", "destination":
			destination = val
		case "opt", "option":
			k, v, _ := strings.Cut(val, "=")
			if k != "" {
				options[k] = v
			}
		default:
			return nil, fmt.Errorf("unexpected key %s in field %s", key, field)
		}
	}
	alias := sourceURI
	return storageStringToSpecConfig(sourceURI, destination, alias, options)
}

func (o *StorageSpecConfigOpt) Set(value string) error {
	storageSpec, err := ParseStorageSpec(value, "/inputs")
	if err != nil {
		return err
	}
	o.values = append(o.values, storageSpec)
	return nil
}

func (o *StorageSpecConfigOpt) Type() string {
	return "storage"
}

func (o *StorageSpecConfigOpt) String() string {
	storages := make([]string, 0, len(o.values))
	for _, storage := range o.values {
		repr := fmt.Sprintf("%s %s %s", storage.Source.Type, storage.Alias, storage.Target)
		storages = append(storages, repr)
	}
	return strings.Join(storages, ", ")
}

func (o *StorageSpecConfigOpt) Values() []*models.InputSource {
	return o.values
}

func storageStringToSpecConfig(sourceURI, destinationPath, alias string, options map[string]string) (*models.InputSource, error) {
	sourceURI = strings.Trim(sourceURI, " '\"")
	destinationPath = strings.Trim(destinationPath, " '\"")
	parsedURI, err := url.Parse(sourceURI)
	if err != nil {
		return nil, err
	}

	var sc *models.SpecConfig
	switch parsedURI.Scheme {
	case "ipfs":
		sc, err = storage_ipfs.NewSpecConfig(parsedURI.Host)
		if err != nil {
			return nil, err
		}
	case "http", "https":
		sc, err = storage_url.NewSpecConfig(sourceURI)
		if err != nil {
			return nil, err
		}
	case "s3":
		s3spec := storage_s3.SourceSpec{}
		s3spec.Bucket = parsedURI.Host
		s3spec.Key = strings.TrimLeft(parsedURI.Path, "/")
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
			case "filter":
				s3spec.Filter = value
			default:
				return nil, fmt.Errorf("unknown option %q for storage %s", key, parsedURI.Scheme)
			}
			if err := s3spec.Validate(); err != nil {
				return nil, err
			}
		}
		sc = &models.SpecConfig{
			Type:   models.StorageSourceS3,
			Params: s3spec.ToMap(),
		}
	case "file":
		source := filepath.Join(parsedURI.Host, parsedURI.Path)
		var rw bool
		for key, value := range options {
			switch key {
			case "rw", "read-write", "read_write", "readwrite":
				readwrite, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return nil, fmt.Errorf("failed to parse read-write option: %s", parseErr)
				}
				rw = readwrite
			default:
				return nil, fmt.Errorf("unknown option %s", key)
			}
		}
		sc, err = storage_local.NewSpecConfig(source, rw)
		if err != nil {
			return nil, err
		}
	case "git", "gitlfs":
		return nil, fmt.Errorf("unsupported type: %s", parsedURI.Scheme)
	default:
		return nil, fmt.Errorf("unknown storage schema: %s", parsedURI.Scheme)
	}

	return &models.InputSource{
		Source: sc,
		Alias:  alias,
		Target: destinationPath,
	}, nil
}
