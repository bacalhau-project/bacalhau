package shared

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"database/sql"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// SQLClient is so we can pass *sql.DB and *sql.Tx to the same functions
type SQLClient interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type GenericSQLDatastore struct {
	mtx              sync.RWMutex
	connectionString string
	db               *sql.DB
}

func NewGenericSQLDatastore(
	db *sql.DB,
	name string,
	connectionString string,
) (*GenericSQLDatastore, error) {
	datastore := &GenericSQLDatastore{
		connectionString: connectionString,
		db:               db,
	}
	datastore.mtx.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        fmt.Sprintf("GenericSQLDatastore[%s].mtx", name),
	})
	return datastore, nil
}

func (d *GenericSQLDatastore) GetDB() *sql.DB {
	return d.db
}

func getJob(db SQLClient, ctx context.Context, id string) (*model.Job, error) {
	var apiversion string
	var jobdata string
	var statedata string
	row := db.QueryRowContext(ctx, `select apiversion, jobdata, statedata from job where id like $1 || '%'`, strings.ToLower(id))
	err := row.Scan(&apiversion, &jobdata, &statedata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, bacerrors.NewJobNotFound(id)
		} else {
			return nil, err
		}
	}
	job, err := model.APIVersionParseJob(apiversion, jobdata)
	if err != nil {
		return nil, err
	}
	state, err := model.APIVersionParseJobState(apiversion, statedata)
	if err != nil {
		return nil, err
	}
	job.Status.State = state
	return &job, nil
}

func (d *GenericSQLDatastore) GetJob(ctx context.Context, id string) (*model.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return getJob(d.db, ctx, id)
}

func getJobsSQL(
	query localdb.JobQuery,
	countMode bool,
) (string, []interface{}, error) {
	var args []interface{}
	clauses := []string{}

	queryCounter := 0
	getQueryCounter := func() string {
		queryCounter++
		return fmt.Sprintf("$%d", queryCounter)
	}

	handleTag := func(annotation string, include bool) {
		appendQuery := " < 1"
		if include {
			appendQuery = " > 0"
		}
		clauses = append(clauses, fmt.Sprintf(`
		(
			select count(*) from job_annotation
			where job_annotation.annotation = %s
			and job_annotation.job_id = job.id
		) %s
		`, getQueryCounter(), appendQuery))
		args = append(args, annotation)
	}

	for _, annotation := range query.IncludeTags {
		handleTag(string(annotation), true)
	}

	for _, annotation := range query.ExcludeTags {
		handleTag(string(annotation), false)
	}

	if query.ClientID != "" {
		clauses = append(clauses, fmt.Sprintf("job.clientid = %s", getQueryCounter()))
		args = append(args, query.ClientID)
	}

	after := ""

	applyOrdering := func(field string) {
		order := "asc"
		if query.SortReverse {
			order = "desc"
		}
		after = after + " order by " + field + " " + order
	}

	if query.SortBy == "created_at" {
		applyOrdering("created")
	} else if query.SortBy == "id" {
		applyOrdering("id")
	} else if query.SortBy != "" {
		return "", nil, fmt.Errorf("invalid sort_by: %s", query.SortBy)
	}

	if query.Limit > 0 {
		after = after + fmt.Sprintf(" limit %d", query.Limit)
	}

	if query.Offset > 0 {
		after = after + fmt.Sprintf(" offset %d", query.Offset)
	}

	where := strings.Join(clauses, " and ")

	if where != "" {
		where = "where " + where
	}

	sql := fmt.Sprintf(`
select
	apiversion,
	jobdata,
	statedata
from
	job
%s
%s
`, where, after)

	if countMode {
		sql = fmt.Sprintf(`
select
	count(job.id) as count
from
	job
%s
%s
`, where, after)
	}

	return sql, args, nil
}

func getJobs(db SQLClient, ctx context.Context, query localdb.JobQuery) ([]*model.Job, error) {
	if query.ID != "" {
		job, err := getJob(db, ctx, query.ID)
		if err != nil {
			return nil, err
		}
		return []*model.Job{job}, nil
	}

	sql, args, err := getJobsSQL(query, false)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []*model.Job{}
	for rows.Next() {
		var innerErr error
		var apiversion string
		var jobdata string
		var statedata string
		var job model.Job
		if innerErr = rows.Scan(&apiversion, &jobdata, &statedata); err != nil {
			return jobs, innerErr
		}
		job, innerErr = model.APIVersionParseJob(apiversion, jobdata)
		if innerErr != nil {
			return nil, err
		}
		state, innerErr := model.APIVersionParseJobState(apiversion, statedata)
		if innerErr != nil {
			return nil, err
		}
		job.Status.State = state
		jobs = append(jobs, &job)
	}
	if err = rows.Err(); err != nil {
		return jobs, err
	}
	return jobs, nil
}

func (d *GenericSQLDatastore) GetJobs(ctx context.Context, query localdb.JobQuery) ([]*model.Job, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return getJobs(d.db, ctx, query)
}

func (d *GenericSQLDatastore) GetJobsCount(ctx context.Context, query localdb.JobQuery) (int, error) {
	if query.ID != "" {
		_, err := getJob(d.db, ctx, query.ID)
		if err != nil {
			return 0, err
		}
		return 1, nil
	}

	useQuery := query
	useQuery.Limit = 0
	useQuery.Offset = 0
	useQuery.SortBy = ""

	sqlQuery, args, err := getJobsSQL(useQuery, true)
	if err != nil {
		return 0, err
	}

	var count int
	row := d.db.QueryRow(sqlQuery, args...)
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func getJobEvents(db SQLClient, ctx context.Context, id string) ([]model.JobEvent, error) {
	var args []interface{}
	args = append(args, id)

	rows, err := db.QueryContext(ctx, `
select
	apiversion,
	eventdata
from
	job_event
where
	job_id = $1
order by
	created asc
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []model.JobEvent
	for rows.Next() {
		var apiversion string
		var eventdata string
		var ev model.JobEvent
		if err = rows.Scan(&apiversion, &eventdata); err != nil {
			return events, err
		}
		ev, err = model.APIVersionParseJobEvent(apiversion, eventdata)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	if err = rows.Err(); err != nil {
		return events, err
	}
	return events, nil
}

func (d *GenericSQLDatastore) GetJobEvents(ctx context.Context, id string) ([]model.JobEvent, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return getJobEvents(d.db, ctx, id)
}

func getJobLocalEvents(db SQLClient, ctx context.Context, id string) ([]model.JobLocalEvent, error) {
	var args []interface{}
	args = append(args, id)

	rows, err := db.QueryContext(ctx, `
select
	apiversion,
	eventdata
from
	local_event
where
	job_id = $1
order by
	created asc
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []model.JobLocalEvent
	for rows.Next() {
		var apiversion string
		var eventdata string
		var ev model.JobLocalEvent
		if err = rows.Scan(&apiversion, &eventdata); err != nil {
			return events, err
		}
		ev, err = model.APIVersionParseJobLocalEvent(apiversion, eventdata)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	if err = rows.Err(); err != nil {
		return events, err
	}
	return events, nil
}

func (d *GenericSQLDatastore) GetJobLocalEvents(ctx context.Context, id string) ([]model.JobLocalEvent, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return getJobLocalEvents(d.db, ctx, id)
}

func (d *GenericSQLDatastore) HasLocalEvent(ctx context.Context, jobID string, eventFilter localdb.LocalEventFilter) (bool, error) {
	jobLocalEvents, err := d.GetJobLocalEvents(ctx, jobID)
	if err != nil {
		return false, err
	}
	hasEvent := false
	for _, localEvent := range jobLocalEvents {
		if eventFilter(localEvent) {
			hasEvent = true
			break
		}
	}
	return hasEvent, nil
}

func (d *GenericSQLDatastore) AddJob(ctx context.Context, j *model.Job) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer tx.Rollback()

	sqlStatement := `
INSERT INTO job (id, created, executor, clientid, apiversion, jobdata)
VALUES ($1, $2, $3, $4, $5, $6)`
	jobData, err := json.Marshal(j)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		sqlStatement,
		j.Metadata.ID,
		j.Metadata.CreatedAt.UTC().Format(time.RFC3339),
		j.Spec.Engine.String(),
		j.Metadata.ClientID,
		model.APIVersionLatest().String(),
		string(jobData),
	)
	if err != nil {
		return err
	}
	for _, annotation := range j.Spec.Annotations {
		sqlStatement := `
INSERT INTO job_annotation (job_id, annotation)
VALUES ($1, $2)`
		_, err = tx.ExecContext(
			ctx,
			sqlStatement,
			j.Metadata.ID,
			annotation,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *GenericSQLDatastore) AddEvent(ctx context.Context, jobID string, ev model.JobEvent) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	//nolint:ineffassign,staticcheck
	sqlStatement := `
INSERT INTO job_event (job_id, created, apiversion, eventdata)
VALUES ($1, $2, $3, $4)`
	eventData, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(
		ctx,
		sqlStatement,
		jobID,
		ev.EventTime.UTC().Format(time.RFC3339),
		model.APIVersionLatest().String(),
		string(eventData),
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *GenericSQLDatastore) AddLocalEvent(ctx context.Context, jobID string, ev model.JobLocalEvent) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	//nolint:ineffassign,staticcheck
	sqlStatement := `
INSERT INTO local_event (job_id, created, apiversion, eventdata)
VALUES ($1, $2, $3, $4)`
	eventData, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(
		ctx,
		sqlStatement,
		jobID,
		time.Now().UTC().Format(time.RFC3339),
		model.APIVersionLatest().String(),
		string(eventData),
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *GenericSQLDatastore) UpdateJobDeal(ctx context.Context, jobID string, deal model.Deal) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	//nolint:ineffassign,staticcheck
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer tx.Rollback()

	job, err := getJob(tx, ctx, jobID)
	if err != nil {
		return err
	}
	job.Spec.Deal = deal
	sqlStatement := `UPDATE JOB SET jobdata = $1, apiversion = $2 WHERE id = $3`
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		sqlStatement,
		string(jobData),
		model.APIVersionLatest().String(),
		jobID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func getJobState(db SQLClient, ctx context.Context, jobID string) (model.JobState, error) {
	var apiversion string
	var statedata string
	row := db.QueryRowContext(ctx, "select apiversion, statedata from job where id = $1 limit 1", jobID)

	err := row.Scan(&apiversion, &statedata)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.JobState{}, fmt.Errorf("job not found: %s %s", jobID, err.Error())
		} else {
			return model.JobState{}, err
		}
	}
	if statedata == "" {
		return model.JobState{
			Nodes: map[string]model.JobNodeState{},
		}, nil
	} else {
		return model.APIVersionParseJobState(apiversion, statedata)
	}
}

func (d *GenericSQLDatastore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	return getJobState(d.db, ctx, jobID)
}

func (d *GenericSQLDatastore) UpdateShardState(
	ctx context.Context,
	jobID, nodeID string,
	shardIndex int,
	update model.JobShardState,
) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer tx.Rollback()

	state, err := getJobState(tx, ctx, jobID)
	if err != nil {
		return err
	}
	err = UpdateShardState(nodeID, shardIndex, &state, update)
	if err != nil {
		return err
	}
	sqlStatement := `UPDATE JOB SET statedata = $1, apiversion = $2 WHERE id = $3`
	stateData, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		sqlStatement,
		string(stateData),
		model.APIVersionLatest().String(),
		jobID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

//go:embed migrations/*.sql
var fs embed.FS

func (d *GenericSQLDatastore) GetMigrations() (*migrate.Migrate, error) {
	files, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, err
	}
	migrations, err := migrate.NewWithSourceInstance("iofs", files, d.connectionString)
	if err != nil {
		return nil, err
	}
	return migrations, nil
}

func (d *GenericSQLDatastore) MigrateUp() error {
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

func (d *GenericSQLDatastore) MigrateDown() error {
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

// Static check to ensure that Transport implements Transport:
var _ localdb.LocalDB = (*GenericSQLDatastore)(nil)
