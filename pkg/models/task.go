package models

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type Task struct {
	// Name of the task
	Name string `json:"Name"`

	Engine *SpecConfig `json:"Engine"`

	Publisher *SpecConfig `json:"Publisher"`

	// Map of environment variables to be used by the driver
	Env map[string]string `json:"Env,omitempty"`

	// Meta is used to associate arbitrary metadata with this task.
	Meta map[string]string `json:"Meta,omitempty"`

	// InputSources is a list of remote artifacts to be downloaded before running the task
	// and mounted into the task.
	InputSources []*InputSource `json:"InputSources,omitempty"`

	// ResultPaths is a list of task volumes to be included in the task's published result
	ResultPaths []*ResultPath `json:"ResultPaths,omitempty"`

	// ResourcesConfig is the resources needed by this task
	ResourcesConfig *ResourcesConfig `json:"Resources,omitempty"`

	Network *NetworkConfig `json:"Network,omitempty"`

	Timeouts *TimeoutConfig `json:"Timeouts,omitempty"`
}

func (t *Task) MetricAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("task_engine", t.Engine.Type),
		attribute.String("task_publisher", t.Publisher.Type),
		attribute.String("task_network", t.Network.Type.String()),
	}
}

func (t *Task) Normalize() {
	// Ensure that an empty and nil map are treated the same
	if t.Meta == nil {
		t.Meta = make(map[string]string)
	}
	if t.Env == nil {
		t.Env = make(map[string]string)
	}
	if t.InputSources == nil {
		t.InputSources = make([]*InputSource, 0)
	}
	if t.ResultPaths == nil {
		t.ResultPaths = make([]*ResultPath, 0)
	}
	if t.ResourcesConfig == nil {
		t.ResourcesConfig = &ResourcesConfig{}
	}
	// publisher is optional and can be empty
	if t.Publisher == nil {
		t.Publisher = &SpecConfig{}
	}
	if t.Network == nil {
		t.Network = &NetworkConfig{}
	}
	if t.Timeouts == nil {
		t.Timeouts = &TimeoutConfig{}
	}
	t.Engine.Normalize()
	t.Publisher.Normalize()
	t.ResourcesConfig.Normalize()
	NormalizeSlice(t.InputSources)
	NormalizeSlice(t.ResultPaths)
	t.Network.Normalize()
	t.ResourcesConfig.Normalize()
}

func (t *Task) Copy() *Task {
	if t == nil {
		return nil
	}
	nt := new(Task)
	*nt = *t
	nt.Engine = t.Engine.Copy()
	nt.Publisher = t.Publisher.Copy()
	nt.ResourcesConfig = t.ResourcesConfig.Copy()
	nt.InputSources = CopySlice(t.InputSources)
	nt.ResultPaths = CopySlice(t.ResultPaths)
	nt.Meta = maps.Clone(t.Meta)
	nt.Env = maps.Clone(t.Env)
	nt.Network = t.Network.Copy()
	nt.Timeouts = t.Timeouts.Copy()
	return nt
}

// Validate is used to check a job for reasonable configuration
func (t *Task) Validate() error {
	var mErr error
	mErr = errors.Join(mErr, t.ValidateSubmission())

	if err := t.Timeouts.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("task timeouts validation failed: %v", err))
	}
	if err := t.ResourcesConfig.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("task resources validation failed: %v", err))
	}
	return mErr
}

// ValidateSubmission is used to check a task for reasonable configuration when it is submitted.
// It is a subset of Validate that does not check fields with defaults, such as timeouts and resources.
func (t *Task) ValidateSubmission() error {
	var mErr error
	if validate.IsBlank(t.Name) {
		mErr = errors.Join(mErr, errors.New("missing task name"))
	} else if validate.ContainsNull(t.Name) {
		mErr = errors.Join(mErr, errors.New("task name contains null character"))
	}
	if err := t.Engine.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("engine validation failed: %v", err))
	}
	if err := t.Publisher.ValidateAllowBlank(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("publisher validation failed: %v", err))
	}
	if err := ValidateSlice(t.InputSources); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("artifact validation failed: %v", err))
	}
	if err := ValidateSlice(t.ResultPaths); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("output validation failed: %v", err))
	}
	if len(t.ResultPaths) > 0 && t.Publisher.IsEmpty() {
		mErr = errors.Join(mErr, errors.New("publisher must be set if result paths are set"))
	}

	seenInputAliases := make(map[string]bool)
	for _, input := range t.InputSources {
		if input.Alias != "" && seenInputAliases[input.Alias] {
			mErr = errors.Join(mErr, fmt.Errorf("input source with alias %s already exist", input.Alias))
		}
		seenInputAliases[input.Alias] = true
	}

	return mErr
}

// ToBuilder returns a new task builder with the same values as the task
func (t *Task) ToBuilder() *TaskBuilder {
	return NewTaskBuilderFromTask(t)
}

func (t *Task) AllStorageTypes() []string {
	var types []string
	for _, a := range t.InputSources {
		types = append(types, a.Source.Type)
	}
	return types
}
