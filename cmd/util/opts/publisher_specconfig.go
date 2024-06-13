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
	publisher_s3 "github.com/bacalhau-project/bacalhau/pkg/s3"
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
		var bucket, key string
		var opts []publisher_s3.PublisherOption
		if _, ok := options["bucket"]; !ok {
			bucket = parsedURI.Host
		} else {
			bucket = options["bucket"]
		}
		if _, ok := options["key"]; !ok {
			key = strings.TrimLeft(parsedURI.Path, "/")
		} else {
			key = options["key"]
		}
		region, ok := options["region"]
		if ok {
			opts = append(opts, publisher_s3.WithPublisherRegion(region))
		}
		endpoint, ok := options["endpoint"]
		if ok {
			opts = append(opts, publisher_s3.WithPublisherEndpoint(endpoint))
		}
		res, err = publisher_s3.NewPublisherSpec(bucket, key, opts...)
		if err != nil {
			return nil, err
		}
	case "local":
		res = publisher_local.NewSpecConfig()
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}
