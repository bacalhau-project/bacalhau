package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/hashicorp/go-multierror"
)

type InputSource struct {
	// Source is the source of the artifact to be downloaded, e.g a URL, S3 bucket, etc.
	Source *SpecConfig

	// Target is the path where the artifact should be mounted on
	Target string
}

// Normalize normalizes the artifact's source and target
func (a *InputSource) Normalize() {
	if a.Source == nil {
		return
	}

	a.Source.Normalize()
	strings.TrimSpace(a.Target)
}

// Copy returns a deep copy of the artifact
func (a *InputSource) Copy() *InputSource {
	if a == nil {
		return nil
	}
	return &InputSource{
		Source: a.Source.Copy(),
		Target: a.Target,
	}
}

// Validate validates the artifact's source and target
func (a *InputSource) Validate() error {
	if a == nil {
		return nil
	}
	var mErr multierror.Error
	if err := a.Source.Validate(); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("invalid artifact source: %w", err))
	}
	if validate.IsBlank(a.Target) {
		mErr.Errors = append(mErr.Errors, errors.New("missing artifact target"))
	}
	return mErr.ErrorOrNil()
}
