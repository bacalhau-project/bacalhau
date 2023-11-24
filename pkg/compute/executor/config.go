package executor

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v3"
)

// config represents the configuration for a single executor plugin
// and is stored in the registry which is a simple lookup by name
type config struct {
	Name string `yaml:"Name"`

	// executable holds the path to the executable for the plugin. If
	// this is a relative path, it is relative to the PLUGIN_HOME.
	Executable string `yaml:"Executable"`

	// network is used to denote whether the plugin prefers to be
	// available over the network. This is used primarily on windows
	// and determines what connection details the plugin is asked to
	// make itself available on.
	Network bool `yaml:"Network"`

	// kind, either 'single' or 'many' specifies whether the plugin
	// will run one execution per process, or many executions per
	// process
	Kind string `yaml:"Type"`

	// maxInstances (optionally) defines the maximum number of processes
	// are allowed for this plugin. This is most likely to be used where
	// kind == single.
	MaxInstances int `yaml:"MaxInstances"`
}

// loadConfig loads the yaml in the provided reader into a config struct,
// or returns an error if it's not valid yaml
func loadConfig(contents []byte) (*config, error) {
	c := new(config)

	if err := yaml.Unmarshal(contents, &c); err != nil {
		return nil, err
	}

	return c, nil
}

// validate checks that the configuration makes sense, and none of
// the variable it contains are invalid
func (c *config) validate(pluginHome string) error {
	errs := new(multierror.Error)

	if c.Executable == "" {
		errs = multierror.Append(errs, fmt.Errorf("path to executable is required"))
	} else {
		// Handle relative files by setting them relative to plugin home
		if !filepath.IsAbs(c.Executable) {
			c.Executable = filepath.Join(pluginHome, c.Executable)
		}

		if executable, reason := isFileExecutable(c.Executable); !executable {
			errs = multierror.Append(errs, errors.New(reason))
		}
	}

	if c.Kind != "single" && c.Kind != "many" {
		errs = multierror.Append(errs, fmt.Errorf("only 'single' or 'many' are allowed for the kind property not '%s'", c.Kind))
	}

	return errs.ErrorOrNil()
}
