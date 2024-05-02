package opts

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	flag "github.com/spf13/pflag"
)

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &PublisherOpt{}

type PublisherOpt struct {
	value *models.SpecConfig
}

func NewPublisherOpt() PublisherOpt {
	return PublisherOpt{value: nil}
}

func (o *PublisherOpt) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
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
	v, err := models.PublisherStringToSpecConfig(destinationURI, options)
	o.value = v
	return err
}

func (o *PublisherOpt) Type() string {
	return "publisher"
}

func (o *PublisherOpt) String() string {
	if o.value == nil {
		return ""
	}
	return o.value.Type
}

func (o *PublisherOpt) Value() *models.SpecConfig {
	if o.value == nil {
		return nil
	}
	return o.value
}
