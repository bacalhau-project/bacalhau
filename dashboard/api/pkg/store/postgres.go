package store

import (
	"context"
	"fmt"

	"embed"
	"time"

	"database/sql"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/types"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type PostgresStore struct {
	mtx              sync.RWMutex
	connectionString string
	db               *sql.DB
}

func NewPostgresStore(
	host string,
	port int,
	database string,
	username string,
	password string,
	autoMigrate bool,
) (*PostgresStore, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", username, password, host, port, database)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{
		connectionString: connectionString,
		db:               db,
	}
	store.mtx.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "PostgresStore.mtx",
	})
	if autoMigrate {
		err = store.MigrateUp()
		if err != nil {
			return nil, fmt.Errorf("there was an error doing the migration: %s", err.Error())
		}
	}
	return store, nil
}

func (d *PostgresStore) LoadUser(
	ctx context.Context,
	username string,
) (*types.User, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var id int
	var created time.Time
	var hashedPassword string
	row := d.db.QueryRow("select id, created, hashed_password from useraccount where username = $1 limit 1", username)
	err := row.Scan(&id, &created, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s %s", username, err.Error())
		} else {
			return nil, err
		}
	}
	return &types.User{
		ID:             id,
		Created:        created,
		Username:       username,
		HashedPassword: hashedPassword,
	}, nil
}

func (d *PostgresStore) LoadUserByID(
	ctx context.Context,
	queryID int,
) (*types.User, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	var username string
	var created time.Time
	var hashedPassword string
	row := d.db.QueryRow("select username, created, hashed_password from useraccount where id = $1 limit 1", queryID)
	err := row.Scan(&username, &created, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s %s", username, err.Error())
		} else {
			return nil, err
		}
	}
	return &types.User{
		ID:             queryID,
		Created:        created,
		Username:       username,
		HashedPassword: hashedPassword,
	}, nil
}

func (d *PostgresStore) GetJobModeration(
	ctx context.Context,
	queryJobID string,
) (*types.JobModeration, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	var id int
	var jobID string
	var userAccountID int
	var created time.Time
	var status string
	var notes string
	row := d.db.QueryRow("select id, job_id, useraccount_id, created, status, notes from job_moderation where job_id = $1 limit 1", queryJobID)
	err := row.Scan(&id, &jobID, &userAccountID, &created, &status, &notes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &types.JobModeration{
		ID:            id,
		JobID:         jobID,
		UserAccountID: userAccountID,
		Created:       created,
		Status:        status,
		Notes:         notes,
	}, nil
}

func (d *PostgresStore) GetAnnotationSummary(
	ctx context.Context,
) ([]*types.AnnotationSummary, error) {
	sqlStatement := `
select
	annotation,
	count(*) as count
from
	job_annotation
group by
	annotation
order by
	annotation
`

	rows, err := d.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []*types.AnnotationSummary{}
	for rows.Next() {
		var annotation string
		var count int
		if err = rows.Scan(&annotation, &count); err != nil {
			return entries, err
		}
		entry := types.AnnotationSummary{
			Annotation: annotation,
			Count:      count,
		}
		entries = append(entries, &entry)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}

func (d *PostgresStore) GetJobMonthSummary(
	ctx context.Context,
) ([]*types.JobMonthSummary, error) {
	sqlStatement := `
select
	concat(
		extract(year from created),
		'-',
		extract(month from created)
	) as month,
	count(*) as count
from
	job
group by
	month
order by
	month
`

	rows, err := d.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []*types.JobMonthSummary{}
	for rows.Next() {
		var month string
		var count int
		if err = rows.Scan(&month, &count); err != nil {
			return entries, err
		}
		entry := types.JobMonthSummary{
			Month: month,
			Count: count,
		}
		entries = append(entries, &entry)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}

func (d *PostgresStore) GetJobExecutorSummary(
	ctx context.Context,
) ([]*types.JobExecutorSummary, error) {
	sqlStatement := `
select
	executor,
	count(*) as count
from
	job
group by
	executor
order by
	executor
`

	rows, err := d.db.Query(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []*types.JobExecutorSummary{}
	for rows.Next() {
		var executor string
		var count int
		if err = rows.Scan(&executor, &count); err != nil {
			return entries, err
		}
		entry := types.JobExecutorSummary{
			Executor: executor,
			Count:    count,
		}
		entries = append(entries, &entry)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, nil
}

func (d *PostgresStore) GetTotalJobsCount(
	ctx context.Context,
) (*types.Counter, error) {
	var count int
	row := d.db.QueryRow("select count(*) as count from job")
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}
	return &types.Counter{
		Count: count,
	}, nil
}

func (d *PostgresStore) GetTotalEventCount(
	ctx context.Context,
) (*types.Counter, error) {
	var count int
	row := d.db.QueryRow("select count(*) as count from job_event")
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}
	return &types.Counter{
		Count: count,
	}, nil
}

func (d *PostgresStore) GetTotalUserCount(
	ctx context.Context,
) (*types.Counter, error) {
	var count int
	row := d.db.QueryRow("select count(distinct clientid) as count from job")
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}
	return &types.Counter{
		Count: count,
	}, nil
}

func (d *PostgresStore) GetTotalExecutorCount(
	ctx context.Context,
) (*types.Counter, error) {
	var count int
	row := d.db.QueryRow("select count(distinct executor) as count from job")
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}
	return &types.Counter{
		Count: count,
	}, nil
}

func (d *PostgresStore) AddUser(
	ctx context.Context,
	username string,
	hashedPassword string,
) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	sqlStatement := `
INSERT INTO useraccount (username, hashed_password)
VALUES ($1, $2)`
	_, err := d.db.Exec(
		sqlStatement,
		username,
		hashedPassword,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *PostgresStore) UpdateUserPassword(
	ctx context.Context,
	username string,
	hashedPassword string,
) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	sqlStatement := `UPDATE useraccount SET hashed_password = $1 WHERE username = $2`
	_, err := d.db.Exec(
		sqlStatement,
		hashedPassword,
		username,
	)
	return err
}

func (d *PostgresStore) CreateJobModeration(
	ctx context.Context,
	moderation types.JobModeration,
) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	sqlStatement := `
INSERT INTO job_moderation (
	job_id,
	useraccount_id,
	status,
	notes
)
VALUES ($1, $2, $3, $4)`
	_, err := d.db.Exec(
		sqlStatement,
		moderation.JobID,
		moderation.UserAccountID,
		moderation.Status,
		moderation.Notes,
	)
	if err != nil {
		return err
	}
	return nil
}

//go:embed migrations/*.sql
var fs embed.FS

func (d *PostgresStore) GetMigrations() (*migrate.Migrate, error) {
	files, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, err
	}
	migrations, err := migrate.NewWithSourceInstance(
		"iofs",
		files,
		fmt.Sprintf("%s&&x-migrations-table=dashboard_schema_migrations", d.connectionString),
	)
	if err != nil {
		return nil, err
	}
	return migrations, nil
}

func (d *PostgresStore) MigrateUp() error {
	migrations, err := d.GetMigrations()
	if err != nil {
		return err
	}
	err = migrations.Up()
	if err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (d *PostgresStore) MigrateDown() error {
	migrations, err := d.GetMigrations()
	if err != nil {
		return err
	}
	err = migrations.Down()
	if err != migrate.ErrNoChange {
		return err
	}
	return nil
}
