package opts

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	flag "github.com/spf13/pflag"
)

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &StorageOpt{}

type StorageOpt struct {
	values []*models.InputSource
}

func (o *StorageOpt) Set(value string) error {
	csvReader := csv.NewReader(strings.NewReader(value))
	fields, err := csvReader.Read()
	if err != nil {
		return err
	}

	var sourceURI string
	destination := "/inputs" // default destination
	options := make(map[string]string)

	for i, field := range fields {
		key, val, ok := strings.Cut(field, "=")

		if !ok {
			// parsing simple format of source:destination
			if i == 0 {
				parsedURI, err := url.Parse(field)
				if err != nil {
					return err
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
				return fmt.Errorf("invalid storage option: %s. Must be a key=value pair", field)
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
			return fmt.Errorf("unpexted key %s in field %s", key, field)
		}
	}
	// TODO(forrest) [correctness]: need to allow aliases to be provided over CLI
	alias := "TODO"
	storageSpec, err := models.StorageStringToSpecConfig(sourceURI, destination, alias, options)
	if err != nil {
		return err
	}
	o.values = append(o.values, storageSpec)
	return nil
}

func (o *StorageOpt) Type() string {
	return "storage"
}

func (o *StorageOpt) String() string {
	storages := make([]string, 0, len(o.values))
	for _, storage := range o.values {
		repr := fmt.Sprintf("%s %s %s", storage.Source.Type, storage.Alias, storage.Target)
		storages = append(storages, repr)
	}
	return strings.Join(storages, ", ")
}

func (o *StorageOpt) Values() []*models.InputSource {
	return o.values
}
