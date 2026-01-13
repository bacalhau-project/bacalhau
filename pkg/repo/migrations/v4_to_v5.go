package migrations

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

const (
	// ProfilesDirName is the directory name for storing CLI profiles.
	ProfilesDirName = "profiles"
	// TokensFileName is the legacy tokens file name.
	TokensFileName = "tokens.json"
	// DefaultProfileName is the name for the default profile.
	DefaultProfileName = "default"
)

// V4ToV5 migrates the repository from version 4 to version 5.
// This migration converts existing tokens.json entries and config.yaml
// client settings into CLI profiles.
func V4ToV5(r repo.FsRepo) error {
	repoPath, err := r.Path()
	if err != nil {
		return fmt.Errorf("getting repo path: %w", err)
	}

	profilesDir := filepath.Join(repoPath, ProfilesDirName)
	store := profile.NewStore(profilesDir)

	// Check if profiles already exist (idempotency)
	existingProfiles, err := store.List()
	if err != nil {
		return fmt.Errorf("listing existing profiles: %w", err)
	}
	if len(existingProfiles) > 0 {
		log.Info().Msgf("Profiles already exist, skipping migration")
		return nil
	}

	var migratedProfiles []string

	// Migrate tokens.json entries
	tokensPath := filepath.Join(repoPath, TokensFileName)
	tokensProfiles, err := migrateTokensFile(store, tokensPath)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to migrate tokens.json, continuing with config.yaml")
	} else {
		migratedProfiles = append(migratedProfiles, tokensProfiles...)
	}

	// Migrate config.yaml client settings
	configProfile, err := migrateConfigFile(r, store)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to migrate config.yaml client settings")
	} else if configProfile != "" {
		migratedProfiles = append(migratedProfiles, configProfile)
	}

	// Set current profile
	if err := setCurrentProfile(store, migratedProfiles); err != nil {
		log.Warn().Err(err).Msg("Failed to set current profile")
	}

	if len(migratedProfiles) > 0 {
		log.Info().Msgf("Migrated %d profiles: %v", len(migratedProfiles), migratedProfiles)
	}

	return nil
}

// migrateTokensFile reads tokens.json and creates profiles for each entry.
// Returns the list of created profile names.
func migrateTokensFile(store *profile.Store, tokensPath string) ([]string, error) {
	file, err := os.Open(tokensPath)
	if os.IsNotExist(err) {
		log.Debug().Msg("tokens.json not found, skipping")
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("opening tokens.json: %w", err)
	}
	defer file.Close()

	// Check if file is empty
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat tokens.json: %w", err)
	}
	if info.Size() == 0 {
		log.Debug().Msg("tokens.json is empty, skipping")
		return nil, nil
	}

	var tokens map[string]string
	if err := json.NewDecoder(file).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("decoding tokens.json: %w", err)
	}

	var createdProfiles []string
	for endpoint, token := range tokens {
		if endpoint == "" {
			continue
		}

		profileName := endpointToProfileName(endpoint)
		p := &profile.Profile{
			Endpoint:    endpoint,
			Description: fmt.Sprintf("Migrated from tokens.json"),
		}
		if token != "" {
			p.Auth = &profile.AuthConfig{
				Token: token,
			}
		}

		if err := store.Save(profileName, p); err != nil {
			log.Warn().Err(err).Str("profile", profileName).Msg("Failed to create profile from token")
			continue
		}

		createdProfiles = append(createdProfiles, profileName)
		log.Debug().Str("profile", profileName).Str("endpoint", endpoint).Msg("Created profile from token")
	}

	return createdProfiles, nil
}

// migrateConfigFile reads config.yaml and creates a default profile from client settings.
// Returns the created profile name, or empty string if no profile was created.
func migrateConfigFile(r repo.FsRepo, store *profile.Store) (string, error) {
	exists, err := configExists(r)
	if err != nil {
		return "", fmt.Errorf("checking config exists: %w", err)
	}
	if !exists {
		log.Debug().Msg("config.yaml not found, skipping")
		return "", nil
	}

	_, cfg, err := readConfig(r)
	if err != nil {
		return "", fmt.Errorf("reading config: %w", err)
	}

	// Check if API settings are configured
	if cfg.API.Host == "" && cfg.API.Port == 0 {
		log.Debug().Msg("No API settings in config.yaml, skipping")
		return "", nil
	}

	// Build endpoint from host and port
	host := cfg.API.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.API.Port
	if port == 0 {
		port = 1234 // Default Bacalhau port
	}

	// Determine scheme based on TLS settings
	scheme := "http"
	if cfg.API.TLS.UseTLS {
		scheme = "https"
	}

	endpoint := fmt.Sprintf("%s://%s:%d", scheme, host, port)

	p := &profile.Profile{
		Endpoint:    endpoint,
		Description: "Migrated from config.yaml",
	}

	// Migrate TLS insecure setting
	if cfg.API.TLS.Insecure {
		p.TLS = &profile.TLSConfig{
			Insecure: true,
		}
	}

	// Check if this profile already exists (from tokens migration)
	if store.Exists(DefaultProfileName) {
		// Merge TLS settings into existing profile
		existing, err := store.Load(DefaultProfileName)
		if err == nil && existing != nil {
			if p.TLS != nil && existing.TLS == nil {
				existing.TLS = p.TLS
			}
			if err := store.Save(DefaultProfileName, existing); err != nil {
				log.Warn().Err(err).Msg("Failed to update existing default profile with TLS settings")
			}
		}
		return "", nil
	}

	if err := store.Save(DefaultProfileName, p); err != nil {
		return "", fmt.Errorf("saving default profile: %w", err)
	}

	log.Debug().Str("profile", DefaultProfileName).Str("endpoint", endpoint).Msg("Created default profile from config")
	return DefaultProfileName, nil
}

// setCurrentProfile sets the current profile, preferring "default" if it exists.
func setCurrentProfile(store *profile.Store, profileNames []string) error {
	if len(profileNames) == 0 {
		return nil
	}

	// Prefer "default" profile
	for _, name := range profileNames {
		if name == DefaultProfileName {
			return store.SetCurrent(DefaultProfileName)
		}
	}

	// Otherwise use the first profile
	return store.SetCurrent(profileNames[0])
}

// endpointToProfileName converts an endpoint URL to a valid profile name.
// Example: "https://prod.example.com:443" -> "prod_example_com_443"
func endpointToProfileName(endpoint string) string {
	// Parse as URL to extract meaningful parts
	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback to simple sanitization
		return sanitizeProfileName(endpoint)
	}

	// Build profile name from host and port
	name := u.Hostname()
	if u.Port() != "" {
		name = name + "_" + u.Port()
	}

	return sanitizeProfileName(name)
}

// sanitizeProfileName converts a string to a valid profile name.
// Replaces invalid characters with underscores.
func sanitizeProfileName(name string) string {
	// Replace common URL separators and invalid characters
	name = strings.ReplaceAll(name, "://", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove consecutive underscores
	re := regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")

	// Trim leading/trailing underscores
	name = strings.Trim(name, "_")

	// Ensure non-empty
	if name == "" {
		return "migrated"
	}

	return name
}
