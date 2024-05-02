package models

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	s3Prefix   = "s3"
	ipfsPrefix = "ipfs"
)

func PublisherStringToSpecConfig(destinationURI string, options map[string]interface{}) (*SpecConfig, error) {
	destinationURI = strings.Trim(destinationURI, " '\"")
	parsedURI, err := url.Parse(destinationURI)
	if err != nil {
		return nil, err
	}

	// handle scenarios where the destinationURI is just the scheme/publisher type, e.g. ipfs
	if parsedURI.Scheme == "" {
		parsedURI.Scheme = parsedURI.Path
	}

	var res *SpecConfig
	switch parsedURI.Scheme {
	case ipfsPrefix:
		res = NewSpecConfig(PublisherIPFS)
	case s3Prefix:
		res = NewSpecConfig(PublisherS3)
		if _, ok := options["bucket"]; !ok {
			res.WithParam("bucket", parsedURI.Host)
		}
		if _, ok := options["key"]; !ok {
			res.WithParam("key", strings.TrimLeft(parsedURI.Path, "/"))
		}
	case "local":
		res = NewSpecConfig(PublisherLocal)
	default:
		return nil, fmt.Errorf("unknown publisher type: %s", parsedURI.Scheme)
	}

	return res, nil
}
