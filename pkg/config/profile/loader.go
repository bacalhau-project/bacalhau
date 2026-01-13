package profile

// Loader handles profile loading with precedence resolution.
type Loader struct {
	store     *Store
	flagValue string // --profile flag value
	envValue  string // BACALHAU_PROFILE env var value
}

// NewLoader creates a new profile loader.
// Precedence: flagValue > envValue > current symlink
func NewLoader(store *Store, flagValue, envValue string) *Loader {
	return &Loader{
		store:     store,
		flagValue: flagValue,
		envValue:  envValue,
	}
}

// Load loads the profile based on precedence rules.
// Returns the profile, profile name, and any error.
// Returns (nil, "", nil) if no profile is selected.
func (l *Loader) Load() (*Profile, string, error) {
	name := l.resolveName()
	if name == "" {
		return nil, "", nil
	}

	p, err := l.store.Load(name)
	if err != nil {
		return nil, "", err
	}

	return p, name, nil
}

// resolveName determines which profile name to use based on precedence.
func (l *Loader) resolveName() string {
	// 1. Flag takes highest precedence
	if l.flagValue != "" {
		return l.flagValue
	}

	// 2. Environment variable
	if l.envValue != "" {
		return l.envValue
	}

	// 3. Current symlink
	// Intentionally ignore errors from GetCurrent() - if the symlink is
	// broken or unreadable, treat it as "no current profile" rather than
	// failing. This is a graceful degradation: the user can always
	// explicitly specify a profile via flag or env var.
	current, _ := l.store.GetCurrent()
	return current
}

// LoadOrCreate loads an existing profile or creates a minimal one.
// Used by SSO flow to bootstrap profiles.
func (l *Loader) LoadOrCreate(name, endpoint string) (*Profile, error) {
	if l.store.Exists(name) {
		return l.store.Load(name)
	}

	// Create minimal profile
	p := &Profile{Endpoint: endpoint}
	if err := l.store.Save(name, p); err != nil {
		return nil, err
	}

	return p, nil
}
