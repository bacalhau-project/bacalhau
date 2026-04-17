package opts

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_ipfs "github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
)

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &PublisherSpecConfigOpt{}

type PublisherSpecConfigOpt struct {
	value *models.SpecConfig
}

func NewPublisherSpecConfigOpt() PublisherSpecConfigOpt {
	return PublisherSpecConfigOpt{value: nil}
}

func (o *PublisherSpecConfigOpt) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	var destinationURI string
	options := make(map[string]string)

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of just publisher type
			if i == 0 {
				destinationURI = field
				continue
			} else {
				return fmt.Errorf("invalid publisher option: %s. Must be a key=value pair", field)
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
			return fmt.Errorf("invalid publisher option: %s", field)
		}
	}
	v, err := publisherStringToSpecConfig(destinationURI, options)
	o.value = v
	return err
}

func (o *PublisherSpecConfigOpt) Type() string {
	return "publisher"
}

func (o *PublisherSpecConfigOpt) String() string {
	if o.value == nil {
		return ""
	}
	return o.value.Type
}

func (o *PublisherSpecConfigOpt) Value() *models.SpecConfig {
	if o.value == nil {
		return nil
	}
	return o.value
}

func publisherStringToSpecConfig(destinationURI string, options map[string]string) (*models.SpecConfig, error) {
	destinationURI = strings.Trim(destinationURI, " '\"")
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
	case "ipfs":
		res = publisher_ipfs.NewSpecConfig()
	case "s3":
		// copy all options to params
		params := map[string]interface{}{}
		for k, v := range options {
			params[k] = v
		}

		// parse bucket and key from URI if not provided in options
		if _, ok := params["bucket"]; !ok {
			params["bucket"] = parsedURI.Host
		}
		if _, ok := params["key"]; !ok {
			params["key"] = strings.TrimLeft(parsedURI.Path, "/")
		}
		return &models.SpecConfig{
			Type:   models.PublisherS3,
			Params: params,
		}, nil
	case "s3managed":
		return &models.SpecConfig{
			Type:   models.PublisherS3Managed,
			Params: make(map[string]interface{}),
		}, nil
	case "local":
		res = publisher_local.NewSpecConfig()
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}
