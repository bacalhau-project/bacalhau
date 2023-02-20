package sqlite

import (
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/filecoin-project/bacalhau/pkg/localdb/shared"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "modernc.org/sqlite"
)

func NewSQLiteDatastore(filename string) (*shared.GenericSQLDatastore, error) {
	db, err := otelsql.Open(
		"sqlite",
		filename,
		otelsql.WithAttributes(semconv.DBSystemSqlite, semconv.PeerService("sqlite")),
	)
	if err != nil {
		return nil, err
	}
	if err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(semconv.DBSystemSqlite)); err != nil { //nolint:govet
		return nil, err
	}
	datastore, err := shared.NewGenericSQLDatastore(
		db,
		"sqlite",
		fmt.Sprintf("sqlite://%s", filename),
	)
	if err != nil {
		return nil, err
	}
	err = datastore.MigrateUp()
	if err != nil {
		return nil, err
	}
	return datastore, err
}
