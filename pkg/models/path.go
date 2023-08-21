package models

import (
	"errors"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/hashicorp/go-multierror"
)

type ResultPath struct {
	// Name
	Name string
	// The path to the file/dir
	Path string
}

// Normalize normalizes the path to a canonical form
func (p *ResultPath) Normalize() {
	if p == nil {
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Path = strings.TrimSpace(p.Path)
}

// Copy returns a copy of the path
func (p *ResultPath) Copy() *ResultPath {
	if p == nil {
		return nil
	}
	return &ResultPath{
		Path: p.Path,
	}
}

// Validate validates the path
func (p *ResultPath) Validate() error {
	if p == nil {
		return errors.New("path is nil")
	}
	var mErr multierror.Error
	if validate.IsBlank(p.Path) {
		mErr.Errors = append(mErr.Errors, errors.New("path is blank"))
	}
	if validate.IsBlank(p.Name) {
		mErr.Errors = append(mErr.Errors, errors.New("name is blank"))
	}
	return mErr.ErrorOrNil()
}
