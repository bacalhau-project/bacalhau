package models

import (
	"errors"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type Path struct {
	// The path to the file/dir
	Location string
}

// Normalize normalizes the path to a canonical form
func (p *Path) Normalize() {
	if p == nil {
		return
	}
	p.Location = strings.TrimSpace(p.Location)
}

// Copy returns a copy of the path
func (p *Path) Copy() *Path {
	if p == nil {
		return nil
	}
	return &Path{
		Location: p.Location,
	}
}

// Validate validates the path
func (p *Path) Validate() error {
	if p == nil {
		return errors.New("path is nil")
	}
	if validate.IsBlank(p.Location) {
		return errors.New("path is blank")
	}
	return nil
}
