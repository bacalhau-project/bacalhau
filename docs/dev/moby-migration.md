# Moby module migration tracker

## Status

**Blocked on upstream.** Not attempted. Holding at the existing `github.com/docker/docker/*` import paths until the ecosystem settles.

## Context

Docker split its canonical Go modules in early 2025:

| Old path | New path |
|---|---|
| `github.com/docker/docker/client` | `github.com/moby/moby/client` |
| `github.com/docker/docker/api/types/*` | `github.com/moby/moby/api/types/*` |
| `github.com/docker/docker/pkg/archive` | `github.com/moby/go-archive` |
| `github.com/docker/docker/pkg/jsonmessage` | `github.com/moby/moby/client/pkg/jsonmessage` |
| `github.com/docker/docker/pkg/stdcopy` | `github.com/moby/moby/api/pkg/stdcopy` |

The VCS path `github.com/docker/docker` now redirects to legacy `github.com/moby/moby v28+incompatible`. The new canonical homes are split submodules (`moby/moby/client`, `moby/moby/api`).

## Why we haven't migrated

Three upstream gaps make a clean cut impossible right now:

1. **`filters` dropped.** `github.com/moby/moby/api` v1.54 does not ship a `filters` package. Our code uses `filters.NewArgs()` across `pkg/executor/docker/`, `pkg/docker/`, and `pkg/compute/logstream/`. There is no published new home for it.

2. **Transitive dep collision.** `test_integration` pulls `buildx`, `docker/compose/v2`, and `testcontainers-go`, all of which still require `docker/docker` (→ legacy `moby/moby v28+incompatible`). If main migrates to the new `moby/moby/api`, Go sees two modules claiming the same `github.com/moby/moby/api/types/*` import paths (legacy module's `/api/types/*` sub-path vs new module's `/types/*` sub-path) and refuses with `ambiguous import`.

3. **Upstream libs mid-split internally.**
   - `docker/buildx` 0.29-0.33: internal file `util/dockerutil/api.go` mixes `docker/docker/client` and `moby/moby/client` imports in the same function → doesn't compile regardless of what we do.
   - `docker/compose/v2` 2.40.3: `cmd/formatter/container.go` mixes `docker/docker/api/types/container.Port` with `moby/moby/api/types/container.PortSummary`.
   - `docker/docker` v28.5.1 & v28.5.2: `pkg/archive/archive_deprecated.go` references `archive.Compression` which no longer exists (moved to `moby/go-archive`).

## When to revisit

Watch for any of these signals:

- `github.com/moby/moby/api` ≥ v1.55 (or similar) that includes a `filters` package — or an upstream migration note documenting where filter construction moved.
- `github.com/docker/buildx` release whose `util/dockerutil/api.go` uses only one of the two client types.
- `github.com/testcontainers/testcontainers-go` release that requires `moby/moby/client` directly (not `docker/docker`).

## Migration steps (when unblocked)

1. Branch from `main`.
2. Replace import strings across `pkg/**/*.go` and `test_integration/**/*.go`:
   ```
   github.com/docker/docker/pkg/archive        → github.com/moby/go-archive
   github.com/docker/docker/pkg/jsonmessage    → github.com/moby/moby/client/pkg/jsonmessage
   github.com/docker/docker/pkg/stdcopy        → github.com/moby/moby/api/pkg/stdcopy
   github.com/docker/docker/api/types          → github.com/moby/moby/api/types
   github.com/docker/docker/client             → github.com/moby/moby/client
   ```
   (Apply longest-paths-first to avoid partial-prefix overlaps. The `api/types` rule catches all subpaths like `/container`, `/mount`, etc.)

3. Resolve `filters` usage — inline or use whatever replaces it.

4. In root `go.mod` and `test_integration/go.mod`:
   - `go mod edit -droprequire=github.com/docker/docker`
   - `go get github.com/moby/moby/client@latest github.com/moby/moby/api@latest github.com/moby/go-archive@latest`
   - `go mod tidy`

5. Bump docker ecosystem deps to whatever post-migration versions exist: `docker/buildx`, `docker/cli`, `docker/compose/v2`, `moby/buildkit`, `testcontainers-go`.

6. Verify: `go build ./...` + `go test -count=1 ./...` in both modules, plus `cd test_integration && go build ./...`.

## Alerts deferred by this

These Dependabot alerts are dismissed as `tolerable_risk` pending this migration (see alert history for current set):

- `github.com/docker/docker` < 29.3.1 (4 alerts on `go.mod` + `test_integration/go.mod`)
- `github.com/moby/buildkit` < 0.28.1 (2 alerts)
- `github.com/docker/cli` < 29.2.0 (1 alert)
- `github.com/containerd/containerd/v2` < 2.1.5 (2 alerts)

When the migration lands, clear the `tolerable_risk` dismissals so any remaining unfixed-in-new-version CVEs resurface.
