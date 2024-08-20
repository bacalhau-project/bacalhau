package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// StagedMigration creates a new migration with staging capabilities.
// It wraps the provided migration function in a staged migration process.
// The migration is a applied to a copy of the repo, iff it's successful the original repo is replaced with the copy
// otherwise the repo remains unchanged.
//
// Parameters:
// - fromVersion: The starting version of the repository
// - toVersion: The target version after migration
// - migrationFn: The function that performs the actual migration
//
// Returns:
// - repo.Migration: A Migration object that encapsulates the staged migration process
func StagedMigration(fromVersion, toVersion int, migrationFn repo.MigrationFn) repo.Migration {
	return repo.NewMigration(
		fromVersion,
		toVersion,
		func(r repo.FsRepo) error {
			return performStagedMigration(r, migrationFn)
		},
	)
}

// performStagedMigration executes the staged migration process.
// It creates a staging area, applies the migration, and then commits the changes.
//
// Parameters:
// - r: The original repository
// - migrationFn: The function that performs the actual migration
//
// Returns:
// - error: Any error encountered during the process
func performStagedMigration(r repo.FsRepo, migrationFn repo.MigrationFn) error {
	repoPath, err := r.Path()
	if err != nil {
		return fmt.Errorf("getting repo path: %w", err)
	}

	log.Info().Msgf("starting staged repo migration for: %q", repoPath)

	stagingPath := filepath.Join(os.TempDir(), fmt.Sprintf("bacalhau-migration-staging-%d", time.Now().UnixNano()))
	if err := os.Mkdir(stagingPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating staging directory %q for repo migration: migration not applied: %w", stagingPath, err)
	}
	defer cleanupStagingPath(stagingPath)

	info, err := os.Stat(stagingPath)
	log.Info().Msgf("BUILD KITE does staging %q exist: INFO: %v, ERR: %v", stagingPath, info, err)
	log.Debug().Msgf("copied repo to staging directory: %q", stagingPath)
	if err := os.CopyFS(stagingPath, os.DirFS(repoPath)); err != nil {
		return fmt.Errorf("copying repository to staging directory %q for repo migration: migration not applied: %w", stagingPath, err)
	}

	stagingRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       stagingPath,
		Migrations: nil,
	})
	if err != nil {
		return fmt.Errorf("creating staging repo for migration: %w", err)
	}

	log.Info().Msgf("running staged repo migration over repo: %q", repoPath)
	if err := migrationFn(*stagingRepo); err != nil {
		return fmt.Errorf("performing migration: %w", err)
	}

	if err := commitMigration(stagingPath, repoPath); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}

	log.Info().Msgf("successfully migrated bacalhau repo %q", repoPath)
	return nil
}

// cleanupStagingPath removes the temporary staging directory.
// It logs a warning if the cleanup fails.
//
// Parameters:
// - stagingPath: The path to the staging directory
func cleanupStagingPath(stagingPath string) {
	if err := os.RemoveAll(stagingPath); err != nil {
		log.Warn().Err(err).Str("path", stagingPath).Msg("failed to clean up staging path for migration.")
	}
}

// commitMigration applies the changes from the staging area to the original repository.
// It creates a backup of the original repo, replaces it with the staged version, and removes the backup if successful.
//
// Parameters:
// - stagingPath: The path to the staging directory
// - repoPath: The path to the original repository
//
// Returns:
// - error: Any error encountered during the process
func commitMigration(stagingPath, repoPath string) error {
	// Replace the original paths with the staged changes
	log.Info().Msgf("committing migration from %q to %q", stagingPath, repoPath)
	// create a backup of the repo pre-migration
	backupPath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf(".bacalhau_backup_%d", time.Now().Unix()))
	log.Info().Msgf("repo backup created in %q", backupPath)
	// move the current bacalhau repo into the backup
	if err := os.Rename(repoPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup bacalhau repo, migration not applied: %w", err)
	}
	// rename move the migrated repo to the specified repo path
	// this is the actual migration
	if err := os.Rename(stagingPath, repoPath); err != nil {
		return fmt.Errorf("failed to migrate repo, migration not applied: %w", err)
	}
	// rename the backup if the above step was successful
	if err := os.RemoveAll(backupPath); err != nil {
		// we don't need to error here as the migration was successful, we just failed to delete the backup which is
		// not critical for bacalhaus operation. A user may manually remove this dir in the rare case this occurs.
		log.Warn().Msgf("successfully applied migration, but failed to remove backup repo at %q: %s", backupPath, err)
		return nil
	}

	return nil
}
