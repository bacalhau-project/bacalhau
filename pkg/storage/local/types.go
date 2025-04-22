package local

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type Source struct {
	SourcePath string
	ReadWrite  bool
	CreateAs   CreateStrategy
}

func (c Source) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (Source, error) {
	if !spec.IsType(models.StorageSourceLocal) && !spec.IsType(models.StorageSourceLocalDirectory) {
		errMsg := fmt.Sprintf(
			"invalid storage source type. expected [%s, %s], but received: %s",
			models.StorageSourceLocal,
			models.StorageSourceLocalDirectory,
			spec.Type,
		)
		return Source{}, errors.New(errMsg)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return Source{}, errors.New("invalid storage source params. cannot be nil")
	}

	// convert readwrite to bool
	if _, ok := inputParams["ReadWrite"]; ok && reflect.TypeOf(inputParams["ReadWrite"]).Kind() == reflect.String {
		inputParams["ReadWrite"] = inputParams["ReadWrite"] == "true"
	}

	var (
		c   Source
		err error
	)
	if err = mapstructure.Decode(spec.Params, &c); err != nil {
		return c, err
	}

	// here c.CreateAs is a string, try to convert it to CreateStrategy
	c.CreateAs, err = CreateStrategyFromString(c.CreateAs.String())
	if err != nil {
		return c, err
	}

	return c, nil
}

func NewSpecConfig(source string, rw bool) (*models.SpecConfig, error) {
	s := Source{
		SourcePath: source,
		ReadWrite:  rw,
	}
	// TODO validate
	return &models.SpecConfig{
		Type:   models.StorageSourceLocal,
		Params: s.ToMap(),
	}, nil
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
