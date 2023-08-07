package models

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/exp/maps"
)

type Task struct {
	// Name of the task
	Name string

	Engine *SpecConfig

	Publisher *SpecConfig

	// Map of environment variables to be used by the driver
	Env map[string]string

	// Meta is used to associate arbitrary metadata with this task.
	Meta map[string]string

	// Artifacts is a list of remote artifacts to be downloaded before running the task
	// and mounted into the task.
	Artifacts []*SpecConfig

	// Volumes is a list of volumes to be mounted into the task.
	Volumes []*SpecConfig

	// Outputs is a list of task volumes to be included in the task's published result
	Outputs []*SpecConfig

	// Resources is the resources needed by this task
	Resources *Resources

	Network *NetworkConfig

	Timeouts *TimeoutConfig
}

func (t *Task) Normalize(*Job) {
	// Ensure that an empty and nil map are treated the same
	if len(t.Meta) == 0 {
		t.Meta = nil
	}
	if len(t.Env) == 0 {
		t.Env = nil
	}
	t.Engine.Normalize()
	t.Publisher.Normalize()
	NormalizeSlice(t.Artifacts)
	NormalizeSlice(t.Volumes)
	NormalizeSlice(t.Outputs)
	t.Network.Normalize()
}

func (t *Task) Copy() *Task {
	if t == nil {
		return nil
	}
	nt := new(Task)
	*nt = *t
	nt.Engine = t.Engine.Copy()
	nt.Publisher = t.Publisher.Copy()
	nt.Resources = t.Resources.Copy()
	nt.Artifacts = CopySlice(t.Artifacts)
	nt.Volumes = CopySlice(t.Volumes)
	nt.Outputs = CopySlice(t.Outputs)
	nt.Meta = maps.Clone(t.Meta)
	nt.Env = maps.Clone(t.Env)
	nt.Network = t.Network.Copy()
	nt.Timeouts = t.Timeouts.Copy()
	return nt
}

func (t *Task) Validate(j *Job) error {
	var mErr multierror.Error
	if validate.IsBlank(t.Name) {
		mErr.Errors = append(mErr.Errors, errors.New("missing task name"))
	} else if validate.ContainsNull(t.Name) {
		mErr.Errors = append(mErr.Errors, errors.New("task name contains null character"))
	}
	if err := t.Engine.Validate(); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("engine validation failed: %v", err))
	}
	if err := t.Publisher.Validate(); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("publisher validation failed: %v", err))
	}
	if err := ValidateSlice(t.Artifacts); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("artifact validation failed: %v", err))
	}
	if err := ValidateSlice(t.Volumes); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("output validation failed: %v", err))
	}
	if err := ValidateSlice(t.Outputs); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("output validation failed: %v", err))
	}
	if err := t.Resources.Validate(); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("task resources validation failed: %v", err))
	}
	if err := t.Timeouts.Validate(); err != nil {
		mErr.Errors = append(mErr.Errors, fmt.Errorf("task timeouts validation failed: %v", err))
	}

	return mErr.ErrorOrNil()
}
