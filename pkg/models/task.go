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

	// Map of environment variables to be used by the driver.
	// Values can be:
	// - Direct value: "debug-mode"
	// - Host env var: "env:HOST_VAR"
	Env map[string]EnvVarValue `json:"Env,omitempty"`

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
		t.Env = make(map[string]EnvVarValue)
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

	if len(t.ResultPaths) > 0 && t.Publisher.IsEmpty() {
		mErr = errors.Join(mErr, errors.New("publisher must be set if result paths are set"))
	}

	if err := t.Timeouts.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("task timeouts validation failed: %v", err))
	}
	return mErr
}

// ValidateSubmission is used to check a task for reasonable configuration when it is submitted.
// It is a subset of Validate that does not check fields with defaults, such as timeouts and resources.
func (t *Task) ValidateSubmission() error {
	mErr := errors.Join(
		validate.NotBlank(t.Name, "missing task name"),
		validate.NoNullChars(t.Name, "task name cannot contain null characters"),
		ValidateEnvVars(t.Env),
	)

	if err := t.Engine.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("engine validation failed: %v", err))
	}
	if err := t.Publisher.ValidateAllowBlank(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("publisher validation failed: %v", err))
	}
	if err := t.Timeouts.ValidateSubmission(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("task timeouts validation failed: %v", err))
	}
	if err := t.ResourcesConfig.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("task resources validation failed: %v", err))
	}
	if err := ValidateSlice(t.InputSources); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("artifact validation failed: %v", err))
	}
	if err := ValidateSlice(t.ResultPaths); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("output validation failed: %v", err))
	}

	if err := t.validateInputSources(); err != nil {
		mErr = errors.Join(mErr, err)
	}

	if err := t.validateResultPaths(); err != nil {
		mErr = errors.Join(mErr, err)
	}

	if err := t.Network.Validate(); err != nil {
		mErr = errors.Join(mErr, fmt.Errorf("network validation failed: %v", err))
	}

	return mErr
}

func (t *Task) validateInputSources() error {
	seenInputAliases := make(map[string]bool)
	seenInputTargets := make(map[string]bool)
	for _, input := range t.InputSources {
		if input.Alias != "" {
			if seenInputAliases[input.Alias] {
				return fmt.Errorf("input source with alias '%s' already exists", input.Alias)
			}
			seenInputAliases[input.Alias] = true
		}
		if input.Target != "" {
			if seenInputTargets[input.Target] {
				return fmt.Errorf("input source with target '%s' already exists", input.Target)
			}
			seenInputTargets[input.Target] = true
		}
	}
	return nil
}

func (t *Task) validateResultPaths() error {
	seenResultNames := make(map[string]bool)
	seenResultPaths := make(map[string]bool)
	for _, result := range t.ResultPaths {
		if result.Name != "" {
			if seenResultNames[result.Name] {
				return fmt.Errorf("result path with name '%s' already exists", result.Name)
			}
			seenResultNames[result.Name] = true
		}
		if result.Path != "" {
			if seenResultPaths[result.Path] {
				return fmt.Errorf("result path '%s' already exists", result.Path)
			}
			seenResultPaths[result.Path] = true
		}
	}
	return nil
}

func (t *Task) AllStorageTypes() []string {
	uniqueTypes := make(map[string]bool)
	for _, a := range t.InputSources {
		uniqueTypes[a.Source.Type] = true
	}

	types := make([]string, 0, len(uniqueTypes))
	for typ := range uniqueTypes {
		types = append(types, typ)
	}
	return types
}
