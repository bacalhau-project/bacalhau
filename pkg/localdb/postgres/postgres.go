package postgres

import (
	"fmt"

	"database/sql"

	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"

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
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
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
			return nil, fmt.Errorf("there was an error doing the migration: %s", err.Error())
		}
	}
	return datastore, err
}
