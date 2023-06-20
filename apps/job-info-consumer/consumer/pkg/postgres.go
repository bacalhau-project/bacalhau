package pkg

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/requester/pubsub/jobinfo"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"
)

type PostgresDatastoreParams struct {
	Host        string
	Port        int
	Database    string
	User        string
	Password    string
	SSLMode     string
	AutoMigrate bool
}

// PostgresDatastore is a postgres datastore.
// It supports migration of the database schema using go-migrate.
type PostgresDatastore struct {
	db     *sql.DB
	dbname string
}

func NewPostgresDatastore(params PostgresDatastoreParams) (*PostgresDatastore, error) {
	sslmode := params.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		params.Host, params.Port, params.User, params.Password, params.Database, sslmode)

	// Open a connection to the database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	datastore := &PostgresDatastore{
		db:     db,
		dbname: params.Database,
	}
	if params.AutoMigrate {
		err = datastore.MigrateUp()
		if err != nil {
			return nil, fmt.Errorf("there was an error doing the migration: %w", err)
		}
	}
	return datastore, nil
}

func (d *PostgresDatastore) Close() error {
	return d.db.Close()
}

func (d *PostgresDatastore) InsertJobInfo(ctx context.Context, envelope jobinfo.Envelope) error {
	// Convert JobWithInfo to JSON
	infoJSON, err := json.Marshal(envelope.Info)
	if err != nil {
		return fmt.Errorf("failed to marshal job info to JSON: %v", err)
	}

	// Perform the insert or update using the ON CONFLICT DO UPDATE clause
	_, err = d.db.Exec(`
INSERT INTO job_info (id, apiversion, info) 
VALUES ($1, $2, $3) 
ON CONFLICT (id) DO UPDATE SET apiversion = $2, info = $3
`, envelope.ID, envelope.APIVersion, infoJSON)
	if err != nil {
		return fmt.Errorf("failed to insert or update job: %v", err)
	}

	log.Info().Msgf("Job with ID %s inserted or updated successfully.", envelope.ID)
	return nil
}

//go:embed migrations/*.sql
var fs embed.FS

func (d *PostgresDatastore) getMigrations() (*migrate.Migrate, error) {
	files, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, err
	}

	drive, err := postgres.WithInstance(d.db, &postgres.Config{})
	if err != nil {
		return nil, err
	}
	return migrate.NewWithInstance("iofs", files, d.dbname, drive)
}

func (d *PostgresDatastore) MigrateUp() error {
	migrations, err := d.getMigrations()
	if err != nil {
		return err
	}
	err = migrations.Up()
	if err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (d *PostgresDatastore) MigrateDown() error {
	migrations, err := d.getMigrations()
	if err != nil {
		return err
	}
	err = migrations.Down()
	if err != migrate.ErrNoChange {
		return err
	}
	return nil
}
