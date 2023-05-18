package store

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
)

type nullableJSON[T any] struct {
	data *T
}

func (v nullableJSON[T]) Scan(src any) error {
	switch data := src.(type) {
	case nil:
		return nil
	case string:
		return json.NewDecoder(strings.NewReader(data)).Decode(&v.data)
	case []byte:
		return json.NewDecoder(bytes.NewReader(data)).Decode(&v.data)
	default:
		return fmt.Errorf("unsupported type %T", src)
	}
}

func (v nullableJSON[T]) Value() (driver.Value, error) {
	if v.data == nil {
		return nil, nil
	} else {
		return json.Marshal(v.data)
	}
}

func JSON[T any](value *T) *nullableJSON[T] {
	return &nullableJSON[T]{data: value}
}

//go:embed queries/*.sql
var queries embed.FS

func SQL(path string) string {
	b, err := queries.ReadFile(filepath.Join("queries", path+".sql"))
	if err != nil {
		panic(err)
	}
	return string(b)
}

type PostgresStore struct {
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

func (d *PostgresStore) GetJobModerations(
	ctx context.Context,
	queryJobID string,
) (results []types.JobModerationSummary, err error) {
	results = make([]types.JobModerationSummary, 0)
	rows, err := d.db.QueryContext(ctx, SQL("get_job_moderations"), queryJobID)
	for err == nil && rows != nil && rows.Next() {
		moderation := new(types.Moderation)
		request := new(types.JobModerationRequest)
		user := new(types.User)

		err = rows.Scan(&moderation.ID, &moderation.RequestID, &moderation.UserAccountID,
			&moderation.Created, &moderation.Status, &moderation.Notes,
			&user.ID, &user.Created, &user.Username,
			&request.ID, &request.JobID, &request.Type,
			&request.Created, &request.Callback)
		if err == nil {
			results = append(results, types.JobModerationSummary{
				Moderation: moderation,
				Request:    request,
				User:       user,
			})
		}
	}
	return results, multierr.Append(err, rows.Err())
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
	moderation types.Moderation,
) error {
	sqlStatement := `
INSERT INTO job_moderation (
	request_id,
	useraccount_id,
	approved,
	notes
)
VALUES ($1, $2, $3, $4)`
	_, err := d.db.ExecContext(
		ctx,
		sqlStatement,
		moderation.RequestID,
		moderation.UserAccountID,
		moderation.Status,
		moderation.Notes,
	)
	return err
}

func (d *PostgresStore) GetModerationRequest(
	ctx context.Context,
	requestID int64,
) (types.ModerationRequest, error) {
	result := new(types.ResultModerationRequest)
	sqlStatement := `
SELECT jmr.id, jmr.job_id, jmr.request_type, jmr.created, jmr.callback,
       rmre.execution_id, rmre.storage_spec
FROM job_moderation_request jmr
LEFT OUTER JOIN result_moderation_request_extra rmre
ON jmr.id = rmre.request_id
WHERE jmr.id = $1;`
	row := d.db.QueryRowContext(ctx, sqlStatement, requestID)
	err := row.Scan(&result.ID, &result.JobID, &result.Type, &result.Created, &result.Callback, JSON(&result.ExecutionID), JSON(&result.StorageSpec))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if result.GetType() == types.ModerationTypeResult {
		return result, nil
	} else {
		return &result.JobModerationRequest, nil
	}
}

func (d *PostgresStore) GetModerationRequestsForJobOfType(
	ctx context.Context,
	jobID string,
	moderationType types.ModerationType,
) ([]types.ModerationRequest, error) {
	results := make([]types.ModerationRequest, 0, 1)
	sqlStatement := `
SELECT jmr.id, jmr.job_id, jmr.request_type, jmr.created, jmr.callback,
       rmre.execution_id, rmre.storage_spec
FROM job_moderation_request jmr
LEFT OUTER JOIN result_moderation_request_extra rmre
ON jmr.id = rmre.request_id
WHERE job_id = $1 AND request_type = $2
ORDER BY created DESC;`
	rows, err := d.db.QueryContext(ctx, sqlStatement, jobID, moderationType)
	for err == nil && rows != nil && rows.Next() {
		result := types.ResultModerationRequest{}
		err = rows.Scan(
			&result.ID,
			&result.JobID,
			&result.Type,
			&result.Created,
			&result.Callback,
			JSON(&result.ExecutionID),
			JSON(&result.StorageSpec),
		)
		if result.GetType() == types.ModerationTypeResult {
			results = append(results, &result)
		} else {
			results = append(results, &result.JobModerationRequest)
		}
	}
	return results, err
}

func (d *PostgresStore) GetModerationRequestsForJob(
	ctx context.Context,
	jobID string,
) (results []types.ModerationRequest, err error) {
	results = make([]types.ModerationRequest, 0, 1)
	sqlStatement := `
SELECT jmr.id, jmr.job_id, jmr.request_type, jmr.created, jmr.callback,
       rmre.execution_id, rmre.storage_spec
FROM job_moderation_request jmr
LEFT OUTER JOIN result_moderation_request_extra rmre
ON jmr.id = rmre.request_id
WHERE job_id = $1
ORDER BY created DESC;`
	rows, err := d.db.QueryContext(ctx, sqlStatement, jobID)
	for err == nil && rows != nil && rows.Next() {
		result := types.ResultModerationRequest{}
		err = rows.Scan(&result.ID, &result.JobID, &result.Type, &result.Created, &result.Callback, JSON(&result.ExecutionID), JSON(&result.StorageSpec))
		if result.GetType() == types.ModerationTypeResult {
			results = append(results, &result)
		} else {
			results = append(results, &result.JobModerationRequest)
		}
	}
	return
}

func (d *PostgresStore) CreateJobModerationRequest(
	ctx context.Context,
	jobID string,
	moderationType types.ModerationType,
	callback *types.URL,
) (result types.ModerationRequest, err error) {
	sqlStatement := `
INSERT INTO job_moderation_request (job_id, request_type, callback)
VALUES ($1, $2, $3);`
	_, err = d.db.ExecContext(ctx, sqlStatement, jobID, moderationType, callback)
	if err != nil {
		return
	}

	requests, err := d.GetModerationRequestsForJobOfType(ctx, jobID, moderationType)
	if len(requests) > 0 {
		result = requests[0]
	}
	return
}

func (d *PostgresStore) CreateResultModerationRequest(
	ctx context.Context,
	executionID model.ExecutionID,
	storageSpec v1beta1.StorageSpec,
	callback *types.URL,
) error {
	jobRequest, err := d.CreateJobModerationRequest(ctx, executionID.JobID, types.ModerationTypeResult, callback)
	if err != nil {
		return err
	}

	sqlStatement := `
INSERT INTO result_moderation_request_extra (request_id, execution_id, storage_spec)
VALUES ($1, $2, $3);`
	_, err = d.db.ExecContext(ctx, sqlStatement, jobRequest.GetID(), JSON(&executionID), JSON(&storageSpec))
	return err
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

func (d *PostgresStore) GetJobsProducingJobInput(ctx context.Context, id string) ([]*types.JobRelation, error) {
	rows, err := d.db.QueryContext(ctx, SQL("get_job_input_relations"), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var relations []*types.JobRelation
	for rows.Next() {
		relation := new(types.JobRelation)
		if err := rows.Scan(&relation.JobID, &relation.CID); err != nil {
			return nil, err
		}
		relations = append(relations, relation)
	}
	return relations, rows.Err()
}

func (d *PostgresStore) GetJobsOperatingOnJobOutput(ctx context.Context, id string) ([]*types.JobRelation, error) {
	rows, err := d.db.QueryContext(ctx, SQL("get_job_output_relations"), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var relations []*types.JobRelation
	for rows.Next() {
		relation := new(types.JobRelation)
		if err := rows.Scan(&relation.JobID, &relation.CID); err != nil {
			return nil, err
		}
		relations = append(relations, relation)
	}
	return relations, rows.Err()
}

func (d *PostgresStore) GetJobsOperatingOnCID(ctx context.Context, data string) ([]*types.JobDataIO, error) {
	rows, err := d.db.QueryContext(ctx, SQL("find_jobs_with_input_or_output"), data)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobIOs []*types.JobDataIO
	for rows.Next() {
		jobIO := new(types.JobDataIO)
		if err := rows.Scan(&jobIO.JobID, &jobIO.InputOutput, &jobIO.IsInput); err != nil {
			return nil, err
		}
		jobIOs = append(jobIOs, jobIO)
	}
	return jobIOs, rows.Err()
}
