package local

import (
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

type CreateStrategy string

const (
	// Create as directory
	Dir CreateStrategy = "dir"

	// Create as file
	File CreateStrategy = "file"

	// Don't create anything
	NoCreate CreateStrategy = "nocreate"
)

const DefaultCreateStrategy = NoCreate
const CreateStrategySpecKey = "CreateAs"

func KnownCreateStrategies() []string {
	return []string{
		Dir.String(),
		File.String(),
		NoCreate.String(),
	}
}

func PermissiveCreateStrategies() []string {
	return []string{
		Dir.String(),
		File.String(),
	}
}

func CreateStrategyFromString(s string) (CreateStrategy, error) {
	switch s {
	case Dir.String():
		return Dir, nil
	case File.String():
		return File, nil
	case NoCreate.String():
		return NoCreate, nil
	case "":
		return DefaultCreateStrategy, nil
	default:
		// TODO: Create a constant for JobSpec to be used in WithComponent for this and similar errors
		return "", bacerrors.Newf("invalid %s value %s", CreateStrategySpecKey, s).
			WithHint("%s must be one of [%s]", CreateStrategySpecKey, strings.Join(KnownCreateStrategies(), ", ")).
			WithCode(bacerrors.ValidationError)
	}
}

func (c CreateStrategy) String() string {
	return string(c)
}
