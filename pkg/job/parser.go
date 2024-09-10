package job

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
)

const (
	s3Prefix    = "s3"
	ipfsPrefix  = "ipfs"
	localPrefix = "local"
)

// ParsePublisherString parses a publisher string into a SpecConfig without having to
// roundtrip through legacy structures.
func ParsePublisherString(publisher string) (*models.SpecConfig, error) {
	publisher = strings.Trim(publisher, " '\"")
	publisher = strings.ToLower(publisher)
	csvReader := csv.NewReader(strings.NewReader(publisher))
	fields, err := csvReader.Read()
	if err != nil {
		return nil, err
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
				return nil, fmt.Errorf("invalid publisher option: %s. Must be a key=value pair", field)
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
			return nil, fmt.Errorf("invalid publisher option: %s", field)
		}
	}

	parsedURI, err := url.Parse(destinationURI)
	if err != nil {
		return nil, err
	}

	// handle scenarios where the destinationURI is just the scheme/publisher type, e.g. ipfs
	if parsedURI.Scheme == "" {
		parsedURI.Scheme = parsedURI.Path
	}

	var res *models.SpecConfig
	switch parsedURI.Scheme {
	case ipfsPrefix:
		res = &models.SpecConfig{
			Type: models.PublisherIPFS,
		}
	case s3Prefix:
		if _, ok := options["bucket"]; !ok {
			options["bucket"] = parsedURI.Host
		}
		if _, ok := options["key"]; !ok {
			options["key"] = strings.TrimLeft(parsedURI.Path, "/")
		}
		res = &models.SpecConfig{
			Type:   models.PublisherS3,
			Params: options,
		}
	case localPrefix:
		res = publisher_local.NewSpecConfig()
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}
