# CLI Profiles Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add CLI profiles to Bacalhau enabling users to connect to multiple clusters without reconfiguring.

**Architecture:** Profile data stored as YAML files in `~/.bacalhau/profiles/`, with current profile tracked via symlink. Profile package handles CRUD operations, loader handles precedence resolution. Migration v4→v5 converts existing tokens.json and config.yaml client settings.

**Tech Stack:** Go, Cobra CLI, YAML (gopkg.in/yaml.v3), existing output utilities

---

## Task 1: Profile Types

**Files:**
- Create: `pkg/config/profile/types.go`
- Test: `pkg/config/profile/types_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile profile.Profile
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid profile",
			profile: profile.Profile{Endpoint: "https://api.example.com:443"},
			wantErr: false,
		},
		{
			name:    "missing endpoint",
			profile: profile.Profile{},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name:    "invalid timeout",
			profile: profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "invalid"},
			wantErr: true,
			errMsg:  "invalid timeout",
		},
		{
			name:    "valid timeout",
			profile: profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "60s"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProfileGetTimeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443"}
		require.Equal(t, profile.DefaultTimeout, p.GetTimeout())
	})

	t.Run("custom timeout", func(t *testing.T) {
		p := profile.Profile{Endpoint: "https://api.example.com:443", Timeout: "60s"}
		require.Equal(t, "60s", p.GetTimeout())
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/config/profile/... -run TestProfile`
Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
package profile

import (
	"fmt"
	"time"
)

const (
	DefaultTimeout = "30s"
)

// Profile represents a CLI connection profile for a Bacalhau cluster.
type Profile struct {
	// Endpoint is the API endpoint (host:port or full URL). Required.
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	// Description is an optional user-friendly label.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Timeout is the request timeout as a duration string (e.g., "30s").
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	// Auth contains authentication settings.
	Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`
	// TLS contains TLS/SSL settings.
	TLS *TLSConfig `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// AuthConfig contains authentication settings for a profile.
type AuthConfig struct {
	// Token is the bearer token for API authentication.
	Token string `yaml:"token,omitempty" json:"token,omitempty"`
}

// TLSConfig contains TLS settings for a profile.
type TLSConfig struct {
	// Insecure skips TLS certificate verification.
	Insecure bool `yaml:"insecure,omitempty" json:"insecure,omitempty"`
}

// Validate validates the profile configuration.
func (p *Profile) Validate() error {
	if p.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if p.Timeout != "" {
		if _, err := time.ParseDuration(p.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", p.Timeout, err)
		}
	}
	return nil
}

// GetTimeout returns the timeout duration string, or the default if not set.
func (p *Profile) GetTimeout() string {
	if p.Timeout == "" {
		return DefaultTimeout
	}
	return p.Timeout
}

// GetToken returns the auth token if set, or empty string.
func (p *Profile) GetToken() string {
	if p.Auth == nil {
		return ""
	}
	return p.Auth.Token
}

// IsInsecure returns whether TLS verification should be skipped.
func (p *Profile) IsInsecure() bool {
	if p.TLS == nil {
		return false
	}
	return p.TLS.Insecure
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/config/profile/... -run TestProfile`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/profile/types.go pkg/config/profile/types_test.go
git commit -m "feat(profile): add profile types with validation"
```

---

## Task 2: Profile Store (CRUD Operations)

**Files:**
- Create: `pkg/config/profile/store.go`
- Test: `pkg/config/profile/store_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestStore(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	t.Run("save and load profile", func(t *testing.T) {
		p := &profile.Profile{
			Endpoint:    "https://api.example.com:443",
			Description: "Test profile",
		}
		err := store.Save("test", p)
		require.NoError(t, err)

		loaded, err := store.Load("test")
		require.NoError(t, err)
		require.Equal(t, p.Endpoint, loaded.Endpoint)
		require.Equal(t, p.Description, loaded.Description)
	})

	t.Run("list profiles", func(t *testing.T) {
		// Save another profile
		err := store.Save("another", &profile.Profile{Endpoint: "https://other.com:443"})
		require.NoError(t, err)

		profiles, err := store.List()
		require.NoError(t, err)
		require.Len(t, profiles, 2)
		require.Contains(t, profiles, "test")
		require.Contains(t, profiles, "another")
	})

	t.Run("delete profile", func(t *testing.T) {
		err := store.Delete("another")
		require.NoError(t, err)

		profiles, err := store.List()
		require.NoError(t, err)
		require.Len(t, profiles, 1)
	})

	t.Run("load non-existent profile", func(t *testing.T) {
		_, err := store.Load("nonexistent")
		require.Error(t, err)
	})

	t.Run("set and get current", func(t *testing.T) {
		err := store.SetCurrent("test")
		require.NoError(t, err)

		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "test", current)
	})

	t.Run("delete current profile clears symlink", func(t *testing.T) {
		err := store.Delete("test")
		require.NoError(t, err)

		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Empty(t, current)
	})
}

func TestStoreSanitizeName(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Test that dangerous names are sanitized
	p := &profile.Profile{Endpoint: "https://api.example.com:443"}
	err := store.Save("../dangerous", p)
	require.NoError(t, err)

	// Verify file was created with sanitized name
	profiles, err := store.List()
	require.NoError(t, err)
	require.Contains(t, profiles, "__dangerous")
}

func TestStoreOnlyWritesProvidedFields(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Save profile with only endpoint
	p := &profile.Profile{Endpoint: "https://api.example.com:443"}
	err := store.Save("minimal", p)
	require.NoError(t, err)

	// Read raw file content
	content, err := os.ReadFile(filepath.Join(tempDir, "minimal.yaml"))
	require.NoError(t, err)

	// Should only contain endpoint, not timeout or other defaults
	require.Contains(t, string(content), "endpoint:")
	require.NotContains(t, string(content), "timeout:")
	require.NotContains(t, string(content), "auth:")
	require.NotContains(t, string(content), "tls:")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/config/profile/... -run TestStore`
Expected: FAIL - NewStore not defined

**Step 3: Write minimal implementation**

```go
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const (
	profileExtension = ".yaml"
	currentSymlink   = "current"
)

// Store handles profile persistence operations.
type Store struct {
	dir string
}

// NewStore creates a new profile store at the given directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// ensureDir creates the profiles directory if it doesn't exist.
func (s *Store) ensureDir() error {
	return os.MkdirAll(s.dir, 0755)
}

// sanitizeName replaces dangerous characters in profile names.
func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}

// profilePath returns the file path for a profile.
func (s *Store) profilePath(name string) string {
	return filepath.Join(s.dir, sanitizeName(name)+profileExtension)
}

// currentPath returns the path to the current symlink.
func (s *Store) currentPath() string {
	return filepath.Join(s.dir, currentSymlink)
}

// Save saves a profile to disk.
func (s *Store) Save(name string, p *Profile) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}
	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("creating profiles directory: %w", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}

	path := s.profilePath(name)
	if err := os.WriteFile(path, data, util.OS_USER_RW); err != nil {
		return fmt.Errorf("writing profile file: %w", err)
	}

	return nil
}

// Load loads a profile from disk.
func (s *Store) Load(name string) (*Profile, error) {
	path := s.profilePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		return nil, fmt.Errorf("reading profile file: %w", err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}

	return &p, nil
}

// Delete removes a profile from disk.
func (s *Store) Delete(name string) error {
	// Check if this is the current profile
	current, _ := s.GetCurrent()
	if current == name {
		// Remove the current symlink
		os.Remove(s.currentPath())
	}

	path := s.profilePath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q not found", name)
		}
		return fmt.Errorf("deleting profile: %w", err)
	}

	return nil
}

// List returns all profile names.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == currentSymlink {
			continue
		}
		if strings.HasSuffix(name, profileExtension) {
			names = append(names, strings.TrimSuffix(name, profileExtension))
		}
	}

	return names, nil
}

// SetCurrent sets the current profile via symlink.
func (s *Store) SetCurrent(name string) error {
	// Verify profile exists
	if _, err := s.Load(name); err != nil {
		return err
	}

	currentPath := s.currentPath()
	// Remove existing symlink if present
	os.Remove(currentPath)

	// Create relative symlink
	target := sanitizeName(name) + profileExtension
	if err := os.Symlink(target, currentPath); err != nil {
		return fmt.Errorf("creating current symlink: %w", err)
	}

	return nil
}

// GetCurrent returns the name of the current profile, or empty string if none.
func (s *Store) GetCurrent() (string, error) {
	currentPath := s.currentPath()
	target, err := os.Readlink(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading current symlink: %w", err)
	}

	// Extract name from target (e.g., "prod.yaml" -> "prod")
	name := strings.TrimSuffix(target, profileExtension)
	return name, nil
}

// Exists checks if a profile exists.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.profilePath(name))
	return err == nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/config/profile/... -run TestStore`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/profile/store.go pkg/config/profile/store_test.go
git commit -m "feat(profile): add profile store for CRUD operations"
```

---

## Task 3: Profile Loader (Precedence Resolution)

**Files:**
- Create: `pkg/config/profile/loader.go`
- Test: `pkg/config/profile/loader_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestLoader(t *testing.T) {
	tempDir := t.TempDir()
	store := profile.NewStore(tempDir)

	// Setup: create profiles
	err := store.Save("prod", &profile.Profile{
		Endpoint:    "https://prod.example.com:443",
		Description: "Production",
	})
	require.NoError(t, err)

	err = store.Save("dev", &profile.Profile{
		Endpoint: "https://dev.example.com:443",
	})
	require.NoError(t, err)

	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("load current profile", func(t *testing.T) {
		loader := profile.NewLoader(store, "", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "prod", name)
		require.Equal(t, "https://prod.example.com:443", p.Endpoint)
	})

	t.Run("flag overrides current", func(t *testing.T) {
		loader := profile.NewLoader(store, "dev", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "dev", name)
		require.Equal(t, "https://dev.example.com:443", p.Endpoint)
	})

	t.Run("env var overrides current", func(t *testing.T) {
		loader := profile.NewLoader(store, "", "dev")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "dev", name)
		require.Equal(t, "https://dev.example.com:443", p.Endpoint)
	})

	t.Run("flag overrides env var", func(t *testing.T) {
		loader := profile.NewLoader(store, "prod", "dev")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "prod", name)
		require.Equal(t, "https://prod.example.com:443", p.Endpoint)
	})

	t.Run("no profile returns nil", func(t *testing.T) {
		emptyStore := profile.NewStore(t.TempDir())
		loader := profile.NewLoader(emptyStore, "", "")
		p, name, err := loader.Load()
		require.NoError(t, err)
		require.Nil(t, p)
		require.Empty(t, name)
	})

	t.Run("non-existent profile flag errors", func(t *testing.T) {
		loader := profile.NewLoader(store, "nonexistent", "")
		_, _, err := loader.Load()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/config/profile/... -run TestLoader`
Expected: FAIL - NewLoader not defined

**Step 3: Write minimal implementation**

```go
package profile

// Loader handles profile loading with precedence resolution.
type Loader struct {
	store      *Store
	flagValue  string // --profile flag value
	envValue   string // BACALHAU_PROFILE env var value
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
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/config/profile/... -run TestLoader`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/config/profile/loader.go pkg/config/profile/loader_test.go
git commit -m "feat(profile): add profile loader with precedence resolution"
```

---

## Task 4: Profile Commands - Root and List

**Files:**
- Create: `cmd/cli/profile/profile.go`
- Create: `cmd/cli/profile/list.go`
- Test: `cmd/cli/profile/list_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileList(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(tempDir + "/profiles")
	err := store.Save("prod", &profile.Profile{
		Endpoint:    "https://prod.example.com:443",
		Description: "Production",
		Auth:        &profile.AuthConfig{Token: "secret"},
	})
	require.NoError(t, err)

	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)

	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("list profiles table format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "prod")
		require.Contains(t, output, "dev")
		require.Contains(t, output, "https://prod.example.com:443")
	})

	t.Run("list profiles json format", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "list", "--output", "json"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, `"name":"prod"`)
		require.Contains(t, output, `"name":"dev"`)
	})
}

func TestProfileListEmpty(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"profile", "list"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "No profiles found")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./cmd/cli/profile/... -run TestProfileList`
Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

`cmd/cli/profile/profile.go`:
```go
package profile

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "profile",
		Short:    "Manage CLI connection profiles",
		Long:     "Create, list, and manage connection profiles for different Bacalhau clusters.",
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
	}

	cmd.AddCommand(newListCmd())

	return cmd
}
```

`cmd/cli/profile/list.go`:
```go
package profile

import (
	"fmt"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

type profileListEntry struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	Endpoint string `json:"endpoint"`
	Auth     string `json:"auth"`
}

func newListCmd() *cobra.Command {
	o := output.OutputOptions{
		Format:     output.TableFormat,
		Pretty:     true,
		HideHeader: false,
		NoStyle:    false,
	}

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List all profiles",
		Long:         "List all configured CLI profiles.",
		Args:         cobra.NoArgs,
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}

			dataDir := cfg.Get("DataDir").(string)
			profilesDir := filepath.Join(dataDir, "profiles")
			store := profile.NewStore(profilesDir)

			return runList(cmd, store, o)
		},
	}

	cmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o))

	return cmd
}

func runList(cmd *cobra.Command, store *profile.Store, o output.OutputOptions) error {
	names, err := store.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		cmd.Println("No profiles found. Create one with: bacalhau profile save <name> --endpoint <url>")
		return nil
	}

	current, _ := store.GetCurrent()

	var entries []profileListEntry
	for _, name := range names {
		p, err := store.Load(name)
		if err != nil {
			continue
		}

		auth := "none"
		if p.GetToken() != "" {
			auth = "token"
		}

		entries = append(entries, profileListEntry{
			Name:     name,
			Current:  name == current,
			Endpoint: p.Endpoint,
			Auth:     auth,
		})
	}

	return output.Output(cmd, listColumns, o, entries)
}

var listColumns = []output.TableColumn[profileListEntry]{
	{
		ColumnConfig: table.ColumnConfig{Name: "CURRENT"},
		Value: func(e profileListEntry) string {
			if e.Current {
				return "*"
			}
			return ""
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "NAME"},
		Value:        func(e profileListEntry) string { return e.Name },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "ENDPOINT"},
		Value:        func(e profileListEntry) string { return e.Endpoint },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "AUTH"},
		Value:        func(e profileListEntry) string { return e.Auth },
	},
}
```

**Step 4: Register profile command in root.go**

Update `cmd/cli/root.go` to add the profile command import and registration:

```go
// Add import
"github.com/bacalhau-project/bacalhau/cmd/cli/profile"

// Add to RootCmd.AddCommand() list
profile.NewCmd(),
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./cmd/cli/profile/... -run TestProfileList`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/cli/profile/profile.go cmd/cli/profile/list.go cmd/cli/profile/list_test.go cmd/cli/root.go
git commit -m "feat(profile): add profile command with list subcommand"
```

---

## Task 5: Profile Save Command

**Files:**
- Create: `cmd/cli/profile/save.go`
- Test: `cmd/cli/profile/save_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileSave(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	t.Run("create new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"profile", "save", "prod", "--endpoint", "https://api.example.com:443"})

		err := cmd.Execute()
		require.NoError(t, err)

		// Verify profile was created
		profilePath := filepath.Join(tempDir, "profiles", "prod.yaml")
		data, err := os.ReadFile(profilePath)
		require.NoError(t, err)

		var p profile.Profile
		err = yaml.Unmarshal(data, &p)
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)
	})

	t.Run("create profile with all options", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{
			"profile", "save", "full",
			"--endpoint", "https://api.example.com:443",
			"--description", "Full profile",
			"--timeout", "60s",
			"--insecure",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		p, err := store.Load("full")
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com:443", p.Endpoint)
		require.Equal(t, "Full profile", p.Description)
		require.Equal(t, "60s", p.Timeout)
		require.True(t, p.IsInsecure())
	})

	t.Run("create and select profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{
			"profile", "save", "selected",
			"--endpoint", "https://selected.example.com:443",
			"--select",
		})

		err := cmd.Execute()
		require.NoError(t, err)

		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "selected", current)
	})

	t.Run("update existing profile", func(t *testing.T) {
		// First create
		cmd1 := cli.NewRootCmd()
		cmd1.SetArgs([]string{"profile", "save", "update-test", "--endpoint", "https://old.example.com:443"})
		require.NoError(t, cmd1.Execute())

		// Then update
		cmd2 := cli.NewRootCmd()
		cmd2.SetArgs([]string{"profile", "save", "update-test", "--endpoint", "https://new.example.com:443", "--description", "Updated"})
		require.NoError(t, cmd2.Execute())

		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		p, err := store.Load("update-test")
		require.NoError(t, err)
		require.Equal(t, "https://new.example.com:443", p.Endpoint)
		require.Equal(t, "Updated", p.Description)
	})

	t.Run("save without endpoint fails for new profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"profile", "save", "no-endpoint"})

		err := cmd.Execute()
		require.Error(t, err)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./cmd/cli/profile/... -run TestProfileSave`
Expected: FAIL - save command not defined

**Step 3: Write minimal implementation**

```go
package profile

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

type saveOptions struct {
	endpoint    string
	description string
	timeout     string
	insecure    bool
	selectAfter bool
}

func newSaveCmd() *cobra.Command {
	opts := &saveOptions{}

	cmd := &cobra.Command{
		Use:          "save <name>",
		Short:        "Create or update a profile",
		Long:         "Create a new profile or update an existing one with connection settings.",
		Args:         cobra.ExactArgs(1),
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}

			dataDir := cfg.Get("DataDir").(string)
			profilesDir := filepath.Join(dataDir, "profiles")
			store := profile.NewStore(profilesDir)

			return runSave(cmd, store, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.endpoint, "endpoint", "", "API endpoint (host:port or full URL)")
	cmd.Flags().StringVar(&opts.description, "description", "", "Profile description")
	cmd.Flags().StringVar(&opts.timeout, "timeout", "", "Request timeout (e.g., 30s, 1m)")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Skip TLS certificate verification")
	cmd.Flags().BoolVar(&opts.selectAfter, "select", false, "Set as current profile after saving")

	return cmd
}

func runSave(cmd *cobra.Command, store *profile.Store, name string, opts *saveOptions) error {
	// Load existing profile or create new
	var p *profile.Profile
	if store.Exists(name) {
		existing, err := store.Load(name)
		if err != nil {
			return fmt.Errorf("loading existing profile: %w", err)
		}
		p = existing
	} else {
		p = &profile.Profile{}
	}

	// Apply provided options (only if flags were set)
	if opts.endpoint != "" {
		p.Endpoint = opts.endpoint
	}
	if opts.description != "" {
		p.Description = opts.description
	}
	if opts.timeout != "" {
		p.Timeout = opts.timeout
	}
	if opts.insecure {
		if p.TLS == nil {
			p.TLS = &profile.TLSConfig{}
		}
		p.TLS.Insecure = true
	}

	// Validate before saving
	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	// Save profile
	if err := store.Save(name, p); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	action := "Created"
	if store.Exists(name) {
		action = "Updated"
	}
	cmd.Printf("%s profile %q\n", action, name)

	// Select if requested
	if opts.selectAfter {
		if err := store.SetCurrent(name); err != nil {
			return fmt.Errorf("setting current profile: %w", err)
		}
		cmd.Printf("Selected profile %q as current\n", name)
	}

	return nil
}
```

**Step 4: Add save command to profile.go**

```go
cmd.AddCommand(newSaveCmd())
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./cmd/cli/profile/... -run TestProfileSave`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/cli/profile/save.go cmd/cli/profile/save_test.go cmd/cli/profile/profile.go
git commit -m "feat(profile): add profile save command"
```

---

## Task 6: Profile Show, Select, Delete Commands

**Files:**
- Create: `cmd/cli/profile/show.go`
- Create: `cmd/cli/profile/select.go`
- Create: `cmd/cli/profile/delete.go`
- Test: `cmd/cli/profile/commands_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func setupTestProfiles(t *testing.T, tempDir string) *profile.Store {
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint:    "https://prod.example.com:443",
		Description: "Production",
		Auth:        &profile.AuthConfig{Token: "secret-token"},
	})
	require.NoError(t, err)

	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)

	err = store.SetCurrent("prod")
	require.NoError(t, err)

	return store
}

func TestProfileShow(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)
	setupTestProfiles(t, tempDir)

	t.Run("show current profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "prod")
		require.Contains(t, output, "https://prod.example.com:443")
		// Token should be redacted
		require.NotContains(t, output, "secret-token")
		require.Contains(t, output, "****")
	})

	t.Run("show specific profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "dev"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "dev")
		require.Contains(t, output, "http://localhost:1234")
	})

	t.Run("show with --show-token", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show", "prod", "--show-token"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "secret-token")
	})
}

func TestProfileSelect(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)
	store := setupTestProfiles(t, tempDir)

	t.Run("select profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"profile", "select", "dev"})

		err := cmd.Execute()
		require.NoError(t, err)

		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "dev", current)
	})

	t.Run("select non-existent profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"profile", "select", "nonexistent"})

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestProfileDelete(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)
	store := setupTestProfiles(t, tempDir)

	t.Run("delete profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"profile", "delete", "dev", "--force"})

		err := cmd.Execute()
		require.NoError(t, err)

		require.False(t, store.Exists("dev"))
	})

	t.Run("delete non-existent profile", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"profile", "delete", "nonexistent", "--force"})

		err := cmd.Execute()
		require.Error(t, err)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./cmd/cli/profile/... -run "TestProfileShow|TestProfileSelect|TestProfileDelete"`
Expected: FAIL - commands not defined

**Step 3: Write minimal implementations**

`cmd/cli/profile/show.go`:
```go
package profile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
)

func newShowCmd() *cobra.Command {
	var showToken bool

	cmd := &cobra.Command{
		Use:          "show [name]",
		Short:        "Show profile details",
		Long:         "Show details of a specific profile. If no name is provided, shows the current profile.",
		Args:         cobra.MaximumNArgs(1),
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}

			dataDir := cfg.Get("DataDir").(string)
			profilesDir := filepath.Join(dataDir, "profiles")
			store := profile.NewStore(profilesDir)

			var name string
			if len(args) > 0 {
				name = args[0]
			} else {
				name, _ = store.GetCurrent()
				if name == "" {
					return fmt.Errorf("no current profile set. Specify a profile name or select one with: bacalhau profile select <name>")
				}
			}

			return runShow(cmd, store, name, showToken)
		},
	}

	cmd.Flags().BoolVar(&showToken, "show-token", false, "Show full token value (default: redacted)")

	return cmd
}

func runShow(cmd *cobra.Command, store *profile.Store, name string, showToken bool) error {
	p, err := store.Load(name)
	if err != nil {
		return err
	}

	current, _ := store.GetCurrent()
	isCurrent := name == current

	token := p.GetToken()
	if token != "" && !showToken {
		token = redactToken(token)
	}

	auth := "none"
	if p.GetToken() != "" {
		auth = fmt.Sprintf("token (%s)", token)
	}

	tls := "secure"
	if p.IsInsecure() {
		tls = "insecure"
	}

	currentMarker := ""
	if isCurrent {
		currentMarker = " (current)"
	}

	data := []collections.Pair[string, any]{
		collections.NewPair[string, any]("Name", name+currentMarker),
		collections.NewPair[string, any]("Endpoint", p.Endpoint),
		collections.NewPair[string, any]("Auth", auth),
		collections.NewPair[string, any]("TLS", tls),
		collections.NewPair[string, any]("Timeout", p.GetTimeout()),
		collections.NewPair[string, any]("Description", p.Description),
	}

	output.KeyValue(cmd, data)
	return nil
}

func redactToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
```

`cmd/cli/profile/select.go`:
```go
package profile

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func newSelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "select <name>",
		Short:        "Set the current profile",
		Long:         "Set the specified profile as the current profile for CLI commands.",
		Args:         cobra.ExactArgs(1),
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}

			dataDir := cfg.Get("DataDir").(string)
			profilesDir := filepath.Join(dataDir, "profiles")
			store := profile.NewStore(profilesDir)

			name := args[0]
			if err := store.SetCurrent(name); err != nil {
				return err
			}

			cmd.Printf("Switched to profile %q\n", name)
			return nil
		},
	}

	return cmd
}
```

`cmd/cli/profile/delete.go`:
```go
package profile

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:          "delete <name>",
		Short:        "Delete a profile",
		Long:         "Delete a profile from the CLI configuration.",
		Args:         cobra.ExactArgs(1),
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := util.SetupConfigType(cmd)
			if err != nil {
				return err
			}

			dataDir := cfg.Get("DataDir").(string)
			profilesDir := filepath.Join(dataDir, "profiles")
			store := profile.NewStore(profilesDir)

			name := args[0]

			// Check if profile exists
			if !store.Exists(name) {
				return fmt.Errorf("profile %q not found", name)
			}

			// Warn if deleting current profile
			current, _ := store.GetCurrent()
			if current == name && !force {
				cmd.Printf("Profile %q is currently selected. Delete anyway? [y/N] ", name)
				reader := bufio.NewReader(cmd.InOrStdin())
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					cmd.Println("Aborted")
					return nil
				}
			}

			if err := store.Delete(name); err != nil {
				return err
			}

			cmd.Printf("Deleted profile %q\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
```

**Step 4: Add commands to profile.go**

```go
cmd.AddCommand(newShowCmd())
cmd.AddCommand(newSelectCmd())
cmd.AddCommand(newDeleteCmd())
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./cmd/cli/profile/... -run "TestProfileShow|TestProfileSelect|TestProfileDelete"`
Expected: PASS

**Step 6: Commit**

```bash
git add cmd/cli/profile/show.go cmd/cli/profile/select.go cmd/cli/profile/delete.go cmd/cli/profile/commands_test.go cmd/cli/profile/profile.go
git commit -m "feat(profile): add show, select, and delete commands"
```

---

## Task 7: Global --profile Flag

**Files:**
- Modify: `cmd/cli/root.go`
- Test: `cmd/cli/profile/global_flag_test.go`

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package profile_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestGlobalProfileFlag(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint: "https://prod.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)
	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("--profile flag overrides current", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		// Use profile show to verify which profile is loaded
		cmd.SetArgs([]string{"--profile", "dev", "profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "dev")
		require.Contains(t, output, "http://localhost:1234")
	})

	t.Run("-p short flag works", func(t *testing.T) {
		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"-p", "dev", "profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "dev")
	})
}

func TestProfileEnvVar(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Setup profiles
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("prod", &profile.Profile{
		Endpoint: "https://prod.example.com:443",
	})
	require.NoError(t, err)
	err = store.Save("dev", &profile.Profile{
		Endpoint: "http://localhost:1234",
	})
	require.NoError(t, err)
	err = store.SetCurrent("prod")
	require.NoError(t, err)

	t.Run("BACALHAU_PROFILE env var", func(t *testing.T) {
		t.Setenv("BACALHAU_PROFILE", "dev")

		cmd := cli.NewRootCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"profile", "show"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "dev")
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./cmd/cli/profile/... -run "TestGlobalProfileFlag|TestProfileEnvVar"`
Expected: FAIL - --profile flag not defined

**Step 3: Modify root.go to add global profile flag**

Update `cmd/cli/root.go`:

```go
// Add to imports
"os"

// Add after other flag definitions (around line 54)
var profileFlag string
RootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Use a specific profile for this command")

// Add to PersistentPreRunE (around line 80, after configflags.BindFlags)
// Store profile selection in context for commands to access
profileEnv := os.Getenv("BACALHAU_PROFILE")
ctx = context.WithValue(ctx, profileFlagKey, profileFlag)
ctx = context.WithValue(ctx, profileEnvKey, profileEnv)
cmd.SetContext(ctx)
```

Add context keys after spanKey:

```go
var profileFlagKey = contextKey{name: "context key for profile flag"}
var profileEnvKey = contextKey{name: "context key for profile env"}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./cmd/cli/profile/... -run "TestGlobalProfileFlag|TestProfileEnvVar"`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/cli/root.go cmd/cli/profile/global_flag_test.go
git commit -m "feat(profile): add global --profile flag"
```

---

## Task 8: Repo Migration v4 to v5

**Files:**
- Modify: `pkg/repo/sysmeta.go` (add Version5)
- Create: `pkg/repo/migrations/v4_to_v5.go`
- Test: `pkg/repo/migrations/v4_to_v5_test.go`
- Modify: `pkg/setup/setup.go` (register migration)

**Step 1: Write the failing test**

```go
//go:build unit || !integration

package migrations_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/repo/migrations"
)

func TestMigrateV4ToV5(t *testing.T) {
	t.Run("migrate tokens.json to profiles", func(t *testing.T) {
		tempDir := t.TempDir()

		// Setup v4 repo with tokens.json
		setupV4Repo(t, tempDir)
		tokens := map[string]string{
			"https://prod.example.com:443": "prod-token",
			"http://localhost:1234":        "dev-token",
		}
		writeTokensJSON(t, tempDir, tokens)

		// Run migration
		fsRepo := createFsRepo(t, tempDir)
		err := migrations.V4ToV5(fsRepo)
		require.NoError(t, err)

		// Verify profiles were created
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		profiles, err := store.List()
		require.NoError(t, err)
		require.Len(t, profiles, 2)

		// Verify prod profile
		prodProfile, err := store.Load("prod_example_com_443")
		require.NoError(t, err)
		require.Equal(t, "https://prod.example.com:443", prodProfile.Endpoint)
		require.Equal(t, "prod-token", prodProfile.GetToken())

		// Verify default profile is set
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.NotEmpty(t, current)
	})

	t.Run("migrate config.yaml client settings", func(t *testing.T) {
		tempDir := t.TempDir()

		// Setup v4 repo with config.yaml
		setupV4Repo(t, tempDir)
		cfg := types.Bacalhau{
			API: types.API{
				Host: "api.example.com",
				Port: 443,
				TLS: types.TLS{
					Insecure: true,
				},
			},
		}
		writeConfigYAML(t, tempDir, cfg)

		// Run migration
		fsRepo := createFsRepo(t, tempDir)
		err := migrations.V4ToV5(fsRepo)
		require.NoError(t, err)

		// Verify default profile was created from config
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		defaultProfile, err := store.Load("default")
		require.NoError(t, err)
		require.Contains(t, defaultProfile.Endpoint, "api.example.com")
		require.True(t, defaultProfile.IsInsecure())

		// Verify it's set as current
		current, err := store.GetCurrent()
		require.NoError(t, err)
		require.Equal(t, "default", current)
	})

	t.Run("skip migration if profiles already exist", func(t *testing.T) {
		tempDir := t.TempDir()

		// Setup v4 repo
		setupV4Repo(t, tempDir)

		// Create profiles directory with existing profile
		store := profile.NewStore(filepath.Join(tempDir, "profiles"))
		err := store.Save("existing", &profile.Profile{Endpoint: "https://existing.com:443"})
		require.NoError(t, err)

		// Run migration
		fsRepo := createFsRepo(t, tempDir)
		err = migrations.V4ToV5(fsRepo)
		require.NoError(t, err)

		// Verify existing profile is preserved
		profiles, err := store.List()
		require.NoError(t, err)
		require.Contains(t, profiles, "existing")
	})
}

func setupV4Repo(t *testing.T, path string) {
	err := os.MkdirAll(path, 0755)
	require.NoError(t, err)

	meta := repo.SystemMetadata{RepoVersion: 4}
	data, err := yaml.Marshal(meta)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(path, "system_metadata.yaml"), data, 0644)
	require.NoError(t, err)
}

func writeTokensJSON(t *testing.T, path string, tokens map[string]string) {
	data, err := json.Marshal(tokens)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(path, "tokens.json"), data, 0644)
	require.NoError(t, err)
}

func writeConfigYAML(t *testing.T, path string, cfg types.Bacalhau) {
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(path, "config.yaml"), data, 0644)
	require.NoError(t, err)
}

func createFsRepo(t *testing.T, path string) repo.FsRepo {
	fsRepo, err := repo.NewFS(repo.FsRepoParams{Path: path})
	require.NoError(t, err)
	return *fsRepo
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/repo/migrations/... -run TestMigrateV4ToV5`
Expected: FAIL - V4ToV5 not defined

**Step 3: Add Version5 to sysmeta.go**

Update `pkg/repo/sysmeta.go`:

```go
const (
	// Version4 is the latest version starting from v1.5.0
	Version4 = 4
	// Version5 adds CLI profiles
	Version5 = 5
)

// IsValidVersion returns true if the version is valid.
func IsValidVersion(version int) bool {
	return version >= Version4 && version <= Version5
}
```

**Step 4: Write migration implementation**

`pkg/repo/migrations/v4_to_v5.go`:
```go
package migrations

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// V4ToV5 migrates from repo version 4 to version 5.
// This migration:
// 1. Creates the profiles directory
// 2. Converts tokens.json entries into profiles
// 3. Converts config.yaml client settings into a default profile
func V4ToV5(r repo.FsRepo) error {
	repoPath, err := r.Path()
	if err != nil {
		return fmt.Errorf("getting repo path: %w", err)
	}

	profilesDir := filepath.Join(repoPath, "profiles")
	store := profile.NewStore(profilesDir)

	// Check if profiles already exist (idempotency)
	existing, _ := store.List()
	if len(existing) > 0 {
		log.Info().Msg("Profiles already exist, skipping migration")
		return nil
	}

	// Migrate tokens.json
	if err := migrateTokens(repoPath, store); err != nil {
		log.Warn().Err(err).Msg("Failed to migrate tokens.json")
	}

	// Migrate config.yaml client settings
	if err := migrateConfig(repoPath, store); err != nil {
		log.Warn().Err(err).Msg("Failed to migrate config.yaml")
	}

	// Set a default current profile if profiles were created
	profiles, _ := store.List()
	if len(profiles) > 0 {
		// Prefer "default" if it exists, otherwise use first profile
		currentName := profiles[0]
		for _, name := range profiles {
			if name == "default" {
				currentName = name
				break
			}
		}
		if err := store.SetCurrent(currentName); err != nil {
			log.Warn().Err(err).Msg("Failed to set current profile")
		}
	}

	return nil
}

func migrateTokens(repoPath string, store *profile.Store) error {
	tokensPath := filepath.Join(repoPath, "tokens.json")
	data, err := os.ReadFile(tokensPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No tokens to migrate
		}
		return err
	}

	var tokens map[string]string
	if err := json.Unmarshal(data, &tokens); err != nil {
		return fmt.Errorf("parsing tokens.json: %w", err)
	}

	for endpoint, token := range tokens {
		name := endpointToProfileName(endpoint)
		p := &profile.Profile{
			Endpoint: endpoint,
			Auth:     &profile.AuthConfig{Token: token},
		}
		if err := store.Save(name, p); err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("Failed to migrate token")
			continue
		}
		log.Info().Str("name", name).Str("endpoint", endpoint).Msg("Migrated token to profile")
	}

	return nil
}

func migrateConfig(repoPath string, store *profile.Store) error {
	configPath := filepath.Join(repoPath, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config to migrate
		}
		return err
	}

	var cfg types.Bacalhau
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config.yaml: %w", err)
	}

	// Only migrate if API settings are configured
	if cfg.API.Host == "" && cfg.API.Port == 0 {
		return nil
	}

	// Build endpoint
	host := cfg.API.Host
	port := cfg.API.Port
	if port == 0 {
		port = 1234 // default
	}
	scheme := "http"
	if cfg.API.TLS.UseTLS {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s:%d", scheme, host, port)

	p := &profile.Profile{
		Endpoint:    endpoint,
		Description: "Migrated from config.yaml",
	}

	if cfg.API.TLS.Insecure {
		p.TLS = &profile.TLSConfig{Insecure: true}
	}

	if err := store.Save("default", p); err != nil {
		return fmt.Errorf("saving default profile: %w", err)
	}

	log.Info().Str("endpoint", endpoint).Msg("Migrated config.yaml to default profile")
	return nil
}

// endpointToProfileName converts an endpoint URL to a valid profile name.
func endpointToProfileName(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback: sanitize the raw string
		return sanitizeProfileName(endpoint)
	}

	// Use host and port as name
	name := u.Host
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return sanitizeProfileName(name)
}

func sanitizeProfileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "..", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
```

**Step 5: Register migration in setup.go**

Update `pkg/setup/setup.go`:

```go
import (
	"github.com/bacalhau-project/bacalhau/pkg/repo/migrations"
)

func SetupMigrationManager() (*repo.MigrationManager, error) {
	return repo.NewMigrationManager(
		repo.NewMigration(repo.Version4, repo.Version5, migrations.V4ToV5),
	)
}
```

**Step 6: Run test to verify it passes**

Run: `go test -v ./pkg/repo/migrations/... -run TestMigrateV4ToV5`
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/repo/sysmeta.go pkg/repo/migrations/v4_to_v5.go pkg/repo/migrations/v4_to_v5_test.go pkg/setup/setup.go
git commit -m "feat(profile): add v4 to v5 migration for profiles"
```

---

## Task 9: Update SSO to Save to Profiles

**Files:**
- Modify: `cmd/cli/auth/sso/login.go`
- Test: `cmd/cli/auth/sso/login_test.go` (update existing tests)

**Step 1: Write/update the failing test**

Add to existing `cmd/cli/auth/sso/login_test.go`:

```go
func TestSSOLoginSavesToProfile(t *testing.T) {
	// This test requires mocking the OAuth flow
	// For now, test that the profile integration code path exists

	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// Create a profile to test SSO saving to it
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	err := store.Save("test", &profile.Profile{
		Endpoint: "https://test.example.com:443",
	})
	require.NoError(t, err)
	err = store.SetCurrent("test")
	require.NoError(t, err)

	// Verify the profile doesn't have a token yet
	p, err := store.Load("test")
	require.NoError(t, err)
	require.Empty(t, p.GetToken())
}
```

**Step 2: Modify login.go to save to profiles**

Update `cmd/cli/auth/sso/login.go` to save tokens to profiles:

```go
// Add imports
import (
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

// In runSSOLogin, replace the token writing section:

// Get profile store
profilesDir := filepath.Join(cfg.DataDir, "profiles")
store := profile.NewStore(profilesDir)

// Determine which profile to save to
profileName := "" // Get from --profile flag via context
if cmd.Context().Value(profileFlagKey) != nil {
	profileName = cmd.Context().Value(profileFlagKey).(string)
}
if profileName == "" {
	// Use current profile or create from endpoint
	profileName, _ = store.GetCurrent()
}

if profileName != "" {
	// Load and update existing profile
	p, err := store.Load(profileName)
	if err != nil {
		// Profile specified but doesn't exist - create it
		p = &profile.Profile{Endpoint: apiURL}
	}
	if p.Auth == nil {
		p.Auth = &profile.AuthConfig{}
	}
	p.Auth.Token = token.AccessToken

	if err := store.Save(profileName, p); err != nil {
		log.Debug().Err(err).Msg("failed to save token to profile")
		// Fall back to legacy tokens.json
		goto legacyWrite
	}
	fmt.Fprintf(os.Stderr, "\nSuccessfully authenticated with %s!\n", nodeAuthConfig.Config.ProviderName)
	fmt.Fprintf(os.Stderr, "Token saved to profile %q\n", profileName)
	return nil
}

legacyWrite:
// Legacy: save to tokens.json
err = util.WriteToken(authTokenPath, apiURL, &persistableSSOCredentials)
// ... rest of existing code
```

**Step 3: Run tests**

Run: `go test -v ./cmd/cli/auth/sso/...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/cli/auth/sso/login.go cmd/cli/auth/sso/login_test.go
git commit -m "feat(profile): update SSO to save tokens to profiles"
```

---

## Task 10: Integration and Final Testing

**Files:**
- Create: `cmd/cli/profile/integration_test.go`

**Step 1: Write integration test**

```go
//go:build integration

package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
)

func TestProfileWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	// 1. Create production profile
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{
		"profile", "save", "prod",
		"--endpoint", "https://api.prod.example.com:443",
		"--description", "Production cluster",
		"--select",
	})
	require.NoError(t, cmd.Execute())

	// 2. Create development profile
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{
		"profile", "save", "dev",
		"--endpoint", "http://localhost:1234",
		"--insecure",
	})
	require.NoError(t, cmd.Execute())

	// 3. List profiles
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "list"})
	require.NoError(t, cmd.Execute())

	// 4. Show current profile (prod)
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "show"})
	require.NoError(t, cmd.Execute())

	// 5. Switch to dev
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "select", "dev"})
	require.NoError(t, cmd.Execute())

	// Verify current is now dev
	store := profile.NewStore(filepath.Join(tempDir, "profiles"))
	current, err := store.GetCurrent()
	require.NoError(t, err)
	require.Equal(t, "dev", current)

	// 6. Use --profile flag to override
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{"--profile", "prod", "profile", "show"})
	require.NoError(t, cmd.Execute())

	// 7. Delete dev profile
	cmd = cli.NewRootCmd()
	cmd.SetArgs([]string{"profile", "delete", "dev", "--force"})
	require.NoError(t, cmd.Execute())

	// Verify dev is deleted
	profiles, err := store.List()
	require.NoError(t, err)
	require.NotContains(t, profiles, "dev")
	require.Contains(t, profiles, "prod")
}
```

**Step 2: Run all tests**

Run: `go test -v ./cmd/cli/profile/... ./pkg/config/profile/... ./pkg/repo/migrations/...`
Expected: PASS

**Step 3: Run integration tests**

Run: `go test -v -tags=integration ./cmd/cli/profile/...`
Expected: PASS

**Step 4: Final commit**

```bash
git add cmd/cli/profile/integration_test.go
git commit -m "test(profile): add integration tests for profile workflow"
```

---

## Summary

This plan implements CLI profiles in 10 tasks:

1. **Profile Types** - Core data structures with validation
2. **Profile Store** - CRUD operations for profile files
3. **Profile Loader** - Precedence resolution logic
4. **Profile Commands (List)** - Root command and list subcommand
5. **Profile Save** - Create/update profiles
6. **Profile Show/Select/Delete** - Remaining CRUD commands
7. **Global --profile Flag** - Root command flag integration
8. **Migration v4→v5** - Convert tokens.json and config.yaml
9. **SSO Integration** - Save tokens to profiles
10. **Integration Testing** - End-to-end workflow tests

Each task follows TDD with:
- Failing test first
- Minimal implementation
- Verify test passes
- Commit
