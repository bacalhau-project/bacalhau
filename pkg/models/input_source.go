package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type InputSource struct {
	// Source is the source of the artifact to be downloaded, e.g a URL, S3 bucket, etc.
	Source *SpecConfig `json:"Source"`

	// Alias is an optional reference to this input source that can be used for
	// dynamic linking to this input. (e.g. dynamic import in wasm by alias)
	Alias string `json:"Alias"`

	// Target is the path where the artifact should be mounted on
	Target string `json:"Target"`
}

func (a *InputSource) MarshalZerologObject(e *zerolog.Event) {
	e.Str("alias", a.Alias).
		Str("target", a.Target).
		Object("source", a.Source)
}

// Normalize normalizes the artifact's source and target
func (a *InputSource) Normalize() {
	if a.Source == nil {
		return
	}

	a.Source.Normalize()
	a.Alias = strings.TrimSpace(a.Alias)
	a.Target = strings.TrimSpace(a.Target)
}

// Copy returns a deep copy of the artifact
func (a *InputSource) Copy() *InputSource {
	if a == nil {
		return nil
	}
	return &InputSource{
		Source: a.Source.Copy(),
		Alias:  a.Alias,
		Target: a.Target,
	}
}

// Validate validates the artifact's source and target
func (a *InputSource) Validate() error {
	if a == nil {
		return nil
	}
	mErr := errors.Join(
		validate.NotBlank(a.Target, "missing artifact target"),
	)
	if err := a.Source.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("invalid artifact source: %w", err))
	}
	return mErr
}
