package registry

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// Registry manages a register mapping plugin names to config structures
// for that plugin.
type Registry struct {
	entries map[string]*Config
}

// NewRegistry creates a new registry to contain the configuration details
// for all of the known plugins
func New() *Registry {
	return &Registry{
		entries: make(map[string]*Config),
	}
}

// Load will inspect the folder defined by pluginHome and after identifying
// configuration files, attempts to load them into the registry against the
// reported name. Once imported data is only ever read from the registry.
func (r *Registry) Load(pluginHome string) error {
	entries, err := os.ReadDir(pluginHome)
	if err != nil {
		return err
	}

	errs := &multierror.Error{}

	for _, f := range entries {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

		fullPath := path.Join(pluginHome, f.Name())

		contents, err := os.ReadFile(fullPath)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		cfg, err := loadConfig(contents)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		// Normalize the name so that we don't accept 'Wasm' and 'wasm' as it will be
		// confusing.
		name := strings.ToLower(cfg.Name)

		// If there's a clash, then we'll complain
		_, found := r.entries[name]
		if found {
			errs = multierror.Append(errs, fmt.Errorf("name '%s' already registered", name))
			continue
		}

		// Validate the config
		if err := cfg.validate(pluginHome); err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		r.entries[name] = cfg
	}

	return errs.ErrorOrNil()
}

func (r *Registry) Get(name string) (*Config, bool) {
	c, b := r.entries[strings.ToLower(name)]
	return c, b
}

// ForEachEntry will iterate over each key-value pair in the registry and
// call fn(k, v) on each pair.  The first time one of these calls returns
// an error then the loop will exit and it will be returned from this
// function.
func (r *Registry) ForEachEntry(fn func(key string, c *Config) error) error {
	for k, v := range r.entries {
		if err := fn(k, v); err != nil {
			return err
		}
	}

	return nil
}
