package postgres

import (
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"
)

func NewPostgresDatastore(
	host string,
	port int,
	database string,
	username string,
	password string,
	autoMigrate bool,
) (*shared.GenericSQLDatastore, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", username, password, host, port, database)
	db, err := otelsql.Open(
		"postgres",
		connectionString,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL, semconv.HostName(host), semconv.PeerService("postgres")),
	)
	if err != nil {
		return nil, err
	}

	if err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(semconv.DBSystemPostgreSQL)); err != nil { //nolint:govet
		return nil, err
	}

	datastore, err := shared.NewGenericSQLDatastore(
		db,
		"postgres",
		connectionString,
	)
	if err != nil {
		return nil, err
	}
	if autoMigrate {
		err = datastore.MigrateUp()
		if err != nil {
			return nil, fmt.Errorf("there was an error doing the migration: %w", err)
		}
	}
	return datastore, err
}
