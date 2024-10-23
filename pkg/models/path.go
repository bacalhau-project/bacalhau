package models

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type ResultPath struct {
	// Name
	Name string `json:"Name"`
	// The path to the file/dir
	Path string `json:"Path"`
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
		Name: p.Name,
		Path: p.Path,
	}
}

// Validate validates the path
func (p *ResultPath) Validate() error {
	if p == nil {
		return errors.New("path is nil")
	}
	return errors.Join(
		validate.NotBlank(p.Path, "missing path"),
		validate.NotBlank(p.Name, "missing name"),
		validate.True(filepath.IsAbs(p.Path), "result path `%s` must be absolute", p.Path),
	)
}
