package opts

import (
	"encoding/csv"
	"fmt"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &PublisherOpt{}

type PublisherOpt struct {
	value v1beta2.PublisherSpec
}

func NewPublisherOptFromSpec(spec v1beta2.PublisherSpec) PublisherOpt {
	return PublisherOpt{value: spec}
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
	o.value, err = parse.ParsePublisherString(destinationURI, options)
	return err
}

func (o *PublisherOpt) Type() string {
	return "publisher"
}

func (o *PublisherOpt) String() string {
	return o.value.Type.String()
}

func (o *PublisherOpt) Value() v1beta2.PublisherSpec {
	return o.value
}
