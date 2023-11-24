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
	entries map[string]*config
}

// NewRegistry creates a new registry to contain the configuration details
// for all of the known plugins
func New() *Registry {
	return &Registry{
		entries: make(map[string]*config),
	}
}

// Load will inspect the folder defined by pluginHome and after identifying
// configuration files, attempts to load them into the registry against the
// reported name.
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
