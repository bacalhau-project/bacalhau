# CLI Profiles Design

## Overview

Add CLI profiles to Bacalhau to enable users to connect to multiple clusters without reconfiguring. Profiles store connection settings, authentication tokens, and client preferences.

## Goals

- Connect to different Bacalhau clusters (dev/staging/prod) without reconfiguring
- Switch between different identities/credentials
- Save configuration presets as reusable profiles
- Maintain backward compatibility through migration

## Storage

### Location

Profiles stored in `~/.bacalhau/profiles/` as individual YAML files:

```
~/.bacalhau/
├── profiles/
│   ├── default.yaml
│   ├── prod.yaml
│   ├── staging.yaml
│   └── current          # symlink to active profile
├── system_metadata.yaml
├── user_id.pem
└── config.yaml          # node config (unchanged)
```

### Profile Format

YAML files with only user-provided fields (no defaults written):

```yaml
# Minimal profile
endpoint: https://api.expanso.io:443
auth:
  token: "eyJ..."
```

```yaml
# Full profile
endpoint: https://api.expanso.io:443
timeout: 60s
description: Production cluster
auth:
  token: "eyJ..."
tls:
  insecure: true
```

### Profile Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `endpoint` | string | Yes | localhost:1234 | API host:port |
| `auth.token` | string | No | - | Bearer token for authentication |
| `tls.insecure` | bool | No | false | Skip TLS certificate verification |
| `timeout` | duration | No | 30s | Request timeout |
| `description` | string | No | - | User-friendly label |

### Current Profile Selection

The current profile is tracked via a symlink at `~/.bacalhau/profiles/current` pointing to the active profile YAML file.

## Commands

### `bacalhau profile list`

List all available profiles.

```bash
$ bacalhau profile list
CURRENT  NAME      ENDPOINT                    AUTH
*        prod      https://api.expanso.io:443  token
         staging   https://staging.expanso.io  token
         default   http://localhost:1234       none
```

Output formats: table (default), JSON, YAML.

### `bacalhau profile show [name]`

Show profile details. Shows current profile if no name provided.

```bash
$ bacalhau profile show prod
Name:        prod
Endpoint:    https://api.expanso.io:443
Auth:        token (tok_****...xyz)
TLS:         secure
Timeout:     30s
Description: Production cluster

$ bacalhau profile show  # shows current
```

Flag `--show-token` to display full token.

### `bacalhau profile save <name>`

Create or update a profile.

```bash
# Create new profile
$ bacalhau profile save prod --endpoint https://api.expanso.io

# Update existing profile
$ bacalhau profile save prod --timeout 60s --description "Production"

# Create and select
$ bacalhau profile save prod --endpoint https://api.expanso.io --select
```

Flags:
- `--endpoint` - API endpoint (host:port or full URL)
- `--timeout` - Request timeout
- `--description` - Profile description
- `--insecure` - Skip TLS verification
- `--select` - Set as current profile after saving

### `bacalhau profile select <name>`

Set the current profile.

```bash
$ bacalhau profile select prod
Switched to profile "prod"
```

### `bacalhau profile delete <name>`

Remove a profile.

```bash
$ bacalhau profile delete staging
Profile "staging" deleted

$ bacalhau profile delete prod
Profile "prod" is currently selected. Delete anyway? [y/N]
```

Flag `--force` to skip confirmation.

## Profile Selection Precedence

When determining which profile to use:

```
1. --profile flag (highest priority)
2. BACALHAU_PROFILE environment variable
3. Current profile symlink (~/.bacalhau/profiles/current)
4. Defaults (localhost:1234, no auth)
```

## Configuration Interaction

### Profiles Replace Client Config

Profiles become the source of truth for client connection settings. After migration:
- Client settings in `config.yaml` are ignored
- `config.yaml` continues to be used for node configuration (compute, orchestrator settings)

### Override Precedence

Within a profile context, settings can still be overridden:

```
1. --config key=value flags (highest, one-off override)
2. BACALHAU_* environment variables
3. Profile values
4. Defaults (lowest)
```

Example:
```bash
# Use prod profile but override port for this command
bacalhau --profile prod --config API.Port=9999 job list
```

### Environment Variables

These environment variables override profile values:
- `BACALHAU_API_HOST` - Override endpoint host
- `BACALHAU_API_PORT` - Override endpoint port
- `BACALHAU_API_KEY` - Override auth token

## Migration (v4 to v5)

### Trigger

Migration runs automatically when opening a repo with version 4.

### Steps

1. Create `~/.bacalhau/profiles/` directory
2. Convert `tokens.json` entries into profiles:
   - Each endpoint becomes a profile
   - Profile named based on endpoint (sanitized)
   - Token stored in `auth.token`
3. Convert client settings from `config.yaml` into `default` profile:
   - `API.Host` and `API.Port` → `endpoint`
   - Other client-relevant settings mapped to profile fields
4. Set `default` profile as current (create symlink)
5. Update repo version to 5

### Backward Compatibility

- Existing users get their settings preserved via migration
- No manual action required
- `tokens.json` can be deleted after migration (or kept as backup)

## SSO Integration

### Current Behavior

`bacalhau auth sso` authenticates against the configured endpoint and saves the token to `tokens.json`.

### New Behavior

SSO saves tokens directly to profiles:

```bash
# Authenticate using current profile's endpoint, save token to current profile
$ bacalhau auth sso

# Authenticate using prod profile's endpoint, save token to prod profile
$ bacalhau auth sso --profile prod
```

### Profile Creation via SSO

If the specified profile doesn't exist, SSO creates it:

```bash
# Creates 'prod' profile using endpoint from env var, authenticates, saves token
$ BACALHAU_API_HOST=api.expanso.io bacalhau auth sso --profile prod
```

The endpoint is resolved from:
1. `--config API.Host` flag
2. `BACALHAU_API_HOST` environment variable
3. Default (localhost:1234)

## Global Flag

Add `--profile` as a global flag on the root command:

```bash
bacalhau --profile prod job list
bacalhau --profile staging node list
```

Short form: `-p`

```bash
bacalhau -p prod job list
```

## Implementation Notes

### Profile Package

Create `pkg/config/profile/` with:
- `types.go` - Profile struct definition
- `store.go` - Load/save/list/delete operations
- `loader.go` - Profile resolution with precedence logic

### Command Package

Create `cmd/cli/profile/` with:
- `root.go` - Profile subcommand
- `list.go` - List command
- `show.go` - Show command
- `save.go` - Save command
- `select.go` - Select command
- `delete.go` - Delete command

### Migration

Add migration to `pkg/repo/migrations/`:
- `v4_to_v5.go` - Migration implementation

Update `pkg/setup/setup.go`:
- Register v4→v5 migration with MigrationManager

### SSO Changes

Update `cmd/cli/auth/sso/login.go`:
- Save token to profile instead of tokens.json
- Support `--profile` flag
- Create profile if missing

### Root Command Changes

Update `cmd/cli/root.go`:
- Add `--profile` / `-p` global flag
- Load profile in `PersistentPreRunE`
- Inject profile config into context

## Security Considerations

- Profile files stored with 0600 permissions (owner read/write only)
- Tokens displayed redacted by default in CLI output
- `--show-token` flag available for debugging/export
- No encryption at rest (users responsible for securing `~/.bacalhau/`)

## Future Considerations

Not in scope for initial implementation:
- Profile export/import commands
- Profile sharing without secrets
- Encrypted token storage
- Certificate-based authentication in profiles