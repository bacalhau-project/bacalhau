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

	stagingPath := filepath.Join(os.TempDir(), fmt.Sprintf("bacalhau-migration-staging-%d", time.Now().UnixNano()))
	if err := os.Mkdir(stagingPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}
	defer cleanupStagingPath(stagingPath)

	log.Info().Str("from", repoPath).Str("to", stagingPath).Msg("migrating repo")
	if err := copyFS(stagingPath, os.DirFS(repoPath)); err != nil {
		return fmt.Errorf("copying repository to staging: %w", err)
	}

	stagingRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       stagingPath,
		Migrations: nil,
	})
	if err != nil {
		return fmt.Errorf("creating staging repo: %w", err)
	}

	if err := migrationFn(*stagingRepo); err != nil {
		return fmt.Errorf("performing migration: %w", err)
	}

	if err := commitMigration(stagingPath, repoPath); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}

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
	log.Info().Msgf("committing staged changes from %q to %q", stagingPath, repoPath)
	// create a backup of the repo pre-migration
	backupPath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf(".bacalhau_backup_%d", time.Now().Unix()))
	// move the current bacalhau repo into the backup
	if err := os.Rename(repoPath, backupPath); err != nil {
		return err
	}
	// rename move the migrated repo to the specified repo path
	// this is the actual migration
	if err := os.Rename(stagingPath, repoPath); err != nil {
		return err
	}
	// rename the backup if the above step was successful
	return os.RemoveAll(backupPath)
}
