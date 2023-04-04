package opts

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	flag "github.com/spf13/pflag"
)

type StorageOpt struct {
	values []model.StorageSpec
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
			if i == 0 {
				sourceURI = field
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
	storageSpec, err := job.ParseStorageString(sourceURI, destination, options)
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
	storages := make([]string, len(o.values))
	for _, storage := range o.values {
		repr := fmt.Sprintf("%s %s %s", storage.StorageSource, storage.Name, storage.Path)
		storages = append(storages, repr)
	}
	return strings.Join(storages, ", ")
}

func (o *StorageOpt) Values() []model.StorageSpec {
	return o.values
}

// compile-time check to ensure type implements the flag.Value interface
var _ flag.Value = &StorageOpt{}
