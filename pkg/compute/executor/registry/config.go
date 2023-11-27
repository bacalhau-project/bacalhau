package registry

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	executor_util "github.com/bacalhau-project/bacalhau/pkg/compute/executor/util"

	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v3"
)

// config represents the configuration for a single executor plugin
// and is stored in the registry which is a simple lookup by name
type Config struct {
	Name string `yaml:"Name"`

	// executable holds the path to the executable for the plugin. If
	// this is a relative path, it is relative to the PLUGIN_HOME.
	Executable string `yaml:"Executable"`

	// Arguments holds any command line parameters to be passed to the
	// plugin.  This allows for plugin options to be passed during
	// startup.
	Arguments []string `yaml:"Arguments"`

	// EnvironmentVariables are passed as K=V pairs to the plugins
	EnvironmentVariables []string `yaml:"EnvironmentVariables"`

	// network is used to denote whether the plugin prefers to be
	// available over the network. This is used primarily on windows
	// and determines what connection details the plugin is asked to
	// make itself available on.
	Network bool `yaml:"Network"`

	// maxInstances (optionally) defines the maximum number of processes
	// are allowed for this plugin. This is most likely to be used where
	// kind == single.
	MaxInstances int `yaml:"MaxInstances"`
}

// loadConfig loads the yaml in the provided reader into a config struct,
// or returns an error if it's not valid yaml
func loadConfig(contents []byte) (*Config, error) {
	c := new(Config)

	if err := yaml.Unmarshal(contents, &c); err != nil {
		return nil, err
	}

	return c, nil
}

// validate checks that the configuration makes sense, and none of
// the variable it contains are invalid
func (c *Config) validate(pluginHome string) error {
	errs := new(multierror.Error)

	if c.Executable == "" {
		errs = multierror.Append(errs, fmt.Errorf("path to executable is required"))
	} else {
		// Handle relative files by setting them relative to plugin home
		if !filepath.IsAbs(c.Executable) {
			c.Executable = filepath.Join(pluginHome, c.Executable)
		}

		if executable, reason := executor_util.IsFileExecutable(c.Executable); !executable {
			errs = multierror.Append(errs, errors.New(reason))
		}
	}

	return errs.ErrorOrNil()
}

func (c *Config) SafeEnvironmentVariables() []string {
	var vars []string

	for _, s := range c.EnvironmentVariables {
		if strings.HasPrefix(s, "BACALHAU_") {
			continue
		}
		vars = append(vars, s)
	}
	return vars
}
