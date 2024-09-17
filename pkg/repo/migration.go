package repo

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type MigrationManager struct {
	migrations map[int]Migration
}

// NewMigrationManager creates a new migration manager.
func NewMigrationManager(migrations ...Migration) (*MigrationManager, error) {
	manager := &MigrationManager{
		migrations: make(map[int]Migration),
	}
	for _, migration := range migrations {
		if err := manager.Add(migration); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

// Add adds a migration to the manager.
func (m *MigrationManager) Add(migration Migration) error {
	if _, ok := m.migrations[migration.FromVersion]; ok {
		return fmt.Errorf("migration from version %d already exists", migration.FromVersion)
	}
	m.migrations[migration.FromVersion] = migration
	return nil
}

// Migrate runs the migrations on the given repo.
func (m *MigrationManager) Migrate(repo FsRepo) error {
	currentVersion, err := repo.Version()
	if err != nil {
		return err
	}
	for {
		migration, ok := m.migrations[currentVersion]
		if !ok {
			break
		}
		log.Info().Msgf("Migrating repo from version %d to %d", migration.FromVersion, migration.ToVersion)
		if err := migration.Migrate(repo); err != nil {
			return err
		}
		currentVersion = migration.ToVersion
		metaStore, err := repo.MetadataStore()
		if err != nil {
			return err
		}
		err = metaStore.WriteVersion(currentVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

type MigrationFn = func(FsRepo) error

type Migration struct {
	FromVersion int
	ToVersion   int
	Migrations  []MigrationFn
}

// NewMigration creates a new migration.
func NewMigration(fromVersion, toVersion int, migrations ...MigrationFn) Migration {
	return Migration{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Migrations:  migrations,
	}
}

// Migrate runs the migrations on the given repo.
func (m Migration) Migrate(repo FsRepo) error {
	for _, migration := range m.Migrations {
		if err := migration(repo); err != nil {
			return err
		}
	}
	return nil
}
