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
	NoCreate CreateStrategy = "noCreate"
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
	switch strings.ToLower(s) {
	case Dir.ToLowerString():
		return Dir, nil
	case File.ToLowerString():
		return File, nil
	case NoCreate.ToLowerString():
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

func (c CreateStrategy) ToLowerString() string {
	return strings.ToLower(c.String())
}
