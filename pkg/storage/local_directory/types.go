package localdirectory

import (
	"errors"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

type Source struct {
	SourcePath string
	ReadWrite  bool
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if spec.Type != models.StorageSourceLocalDirectory {
		return Source{}, errors.New("invalid storage source type. expected " + models.StorageSourceLocalDirectory + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return Source{}, errors.New("invalid storage source params. cannot be nil")
	}

	// convert readwrite to bool
	if _, ok := inputParams["ReadWrite"]; ok && reflect.TypeOf(inputParams["ReadWrite"]).Kind() == reflect.String {
		inputParams["ReadWrite"] = inputParams["ReadWrite"] == "true"
	}

	var c Source
	if err := mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	return c, nil
}

type AllowedPath struct {
	Path      string
	ReadWrite bool
}

// string representation of the object.
func (obj AllowedPath) String() string {
	suffix := "ro"
	if obj.ReadWrite {
		suffix = "rw"
	}
	return obj.Path + ":" + suffix
}

func ParseAllowPath(path string) AllowedPath {
	if strings.HasSuffix(path, ":rw") {
		return AllowedPath{
			Path:      strings.TrimSuffix(path, ":rw"),
			ReadWrite: true,
		}
	} else if strings.HasSuffix(path, ":ro") {
		return AllowedPath{
			Path:      strings.TrimSuffix(path, ":ro"),
			ReadWrite: false,
		}
	} else {
		return AllowedPath{
			Path:      path,
			ReadWrite: false,
		}
	}
}

func ParseAllowPaths(paths []string) []AllowedPath {
	allowedPaths := make([]AllowedPath, len(paths))
	for i, path := range paths {
		allowedPaths[i] = ParseAllowPath(path)
	}
	return allowedPaths
}
