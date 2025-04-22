package local

import (
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/rs/zerolog/log"
)

type CreateStrategy string

const (
	// Try to infer the type (file vs directory) from path
	Infer CreateStrategy = "infer"

	// Create as directory
	Dir CreateStrategy = "dir"

	// Create as file
	File CreateStrategy = "file"

	// Don't create anything
	NoCreate CreateStrategy = "nocreate"
)

const DefaultCreateStrategy = Infer
const CreateStrategySpecKey = "CreateAs"

func AllowedCreateStrategies() []string {
	return []string{
		Infer.String(),
		Dir.String(),
		File.String(),
		NoCreate.String(),
	}
}

func CreateStrategyFromString(s string) (CreateStrategy, error) {
	switch s {
	case Infer.String():
		return Infer, nil
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
		return "", bacerrors.Newf("invalid CreateAs value %s", s).
			WithHint("CreateAs must be one of [%s]", strings.Join(AllowedCreateStrategies(), ", ")).
			WithCode(bacerrors.ValidationError)
	}
}

// Attempts to infer whether the given path represents a file or directory.
// This is a best-effort attempt based on common conventions.
func InferCreateStrategyFromPath(path string) CreateStrategy {
	// Leverage filepath.Split, it handles edge cases like trailing slashes, no slashes, etc.
	// For now this is smart enough, but we can improve it later if needed.
	// Note that there are some noticeable exceptions for which this will return the wrong strategy,
	// For example folders with a dot in the name (e.g. /etc/conf.d) and without a trailing slash
	// will be considered files. However such paths are likely to be non-empty and CreateStrategy will not be called for them.
	// Additinally, we can look at Target value to see if it gives more insight on whether we should create a file or directory.
	_, file := filepath.Split(path)
	var inferredStrategy CreateStrategy
	if file == "" {
		inferredStrategy = Dir
	} else {
		inferredStrategy = File
	}
	log.Debug().Str("path", path).Msgf("inferred create strategy: %s", inferredStrategy)
	return inferredStrategy
}

func (c CreateStrategy) String() string {
	return string(c)
}
