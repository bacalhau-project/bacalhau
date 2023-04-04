package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgtype"
	"github.com/raulk/clock"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewStore(opts ...ConfigOpt) (*Store, error) {
	cfg := NewDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	db, err := gorm.Open(cfg.Dialect, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdlConns)

	if err := db.AutoMigrate(&Job{}, &JobState{}, &ExecutionState{}); err != nil {
		return nil, err
	}
	return &Store{Db: db, clock: cfg.Clock}, nil
}

var _ jobstore.Store = (*Store)(nil)

type Store struct {
	Db    *gorm.DB
	clock clock.Clock
}

func (s *Store) CreateJob(ctx context.Context, j model.Job) error {
	if has, err := s.hasJob(ctx, j.ID()); err != nil {
		return err
	} else if has {
		return jobstore.NewErrJobAlreadyExists(j.ID())
	}

	jb, err := json.Marshal(j)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	jobModel := Job{
		JobID: j.ID(),
		Job: pgtype.JSONB{
			Bytes:  jb,
			Status: pgtype.Present,
		},
		CreatedAt: now,
	}
	jobStateModel := JobState{
		JobID:         j.ID(),
		Version:       1,
		CurrentState:  int(model.JobStateNew),
		PreviousState: int(model.JobStateNew),
		Comment:       "Job Created",
		CreatedAt:     now,
	}
	return s.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&jobModel).Error; err != nil {
			return err
		}

		if err := tx.Create(&jobStateModel).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Store) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) error {
	jobState, err := s.GetJobState(ctx, request.JobID)
	if err != nil {
		return err
	}

	if err := request.Condition.Validate(jobState); err != nil {
		return err
	}

	if jobState.State.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(jobState.JobID, jobState.State, request.NewState)
	}

	newJobStateModel := &JobState{
		JobID:         jobState.JobID,
		Version:       jobState.Version + 1,
		CurrentState:  int(request.NewState),
		PreviousState: int(jobState.State),
		Comment:       request.Comment,
		CreatedAt:     s.clock.Now(),
	}
	return s.Db.Create(newJobStateModel).Error
}

func (s *Store) GetJob(ctx context.Context, id string) (model.Job, error) {
	// TODO remove when https://github.com/bacalhau-project/bacalhau/issues/2298 lands
	if len(id) < model.ShortIDLength {
		return model.Job{}, jobstore.NewErrJobNotFound(id)
	}
	if jobutils.ShortID(id) == id {
		return model.Job{}, fmt.Errorf("short JobIDs are not supported")
	}

	var jobModel Job
	res := s.Db.WithContext(ctx).
		Limit(1).
		Find(&jobModel, "job_id = ?", id)
	if res.Error != nil {
		return model.Job{}, res.Error
	}
	if res.RowsAffected == 0 {
		return model.Job{}, jobstore.NewErrJobNotFound(id)
	}
	return jobModel.AsJob()
}

func (s *Store) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
	if query.ID != "" {
		j, err := s.GetJob(ctx, query.ID)
		if err != nil {
			return nil, err
		}
		return []model.Job{j}, nil
	}
	// TODO write an actual query
	// - not doing this now since the Job Model contains model.Job as json and json queries are not consisten across Db impls.
	jobs, err := s.getAllJobs(ctx)
	if err != nil {
		return nil, err
	}
	var result []model.Job
	for _, j := range jobs {
		if query.Limit > 0 && len(result) == query.Limit {
			break
		}

		if !query.ReturnAll && query.ClientID != "" && query.ClientID != j.Metadata.ClientID {
			// Job is not for the requesting client, so ignore it.
			continue
		}

		// If we are not using include tags, by default every job is included.
		// If a job is specifically included, that overrides it being excluded.
		included := len(query.IncludeTags) == 0
		for _, tag := range j.Spec.Annotations {
			if slices.Contains(query.IncludeTags, model.IncludedTag(tag)) {
				included = true
				break
			}
			if slices.Contains(query.ExcludeTags, model.ExcludedTag(tag)) {
				included = false
				break
			}
		}

		if !included {
			continue
		}

		result = append(result, j)
	}

	listSorter := func(i, j int) bool {
		switch query.SortBy {
		case "id":
			if query.SortReverse {
				// what does it mean to sort by ID?
				return result[i].Metadata.ID > result[j].Metadata.ID
			} else {
				return result[i].Metadata.ID < result[j].Metadata.ID
			}
		case "created_at":
			if query.SortReverse {
				return result[i].Metadata.CreatedAt.UTC().Unix() > result[j].Metadata.CreatedAt.UTC().Unix()
			} else {
				return result[i].Metadata.CreatedAt.UTC().Unix() < result[j].Metadata.CreatedAt.UTC().Unix()
			}
		default:
			return false
		}
	}
	sort.Slice(result, listSorter)
	return result, nil
}

func (s *Store) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	jobStateModel, err := s.getLatestJobStateModel(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}

	executions, err := s.getLatestExecutions(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}
	return model.JobState{
		JobID:      jobID,
		Executions: executions,
		State:      model.JobStateType(jobStateModel.CurrentState),
		Version:    jobStateModel.Version,
		CreateTime: job.Metadata.CreatedAt,
		UpdateTime: jobStateModel.CreatedAt,
		TimeoutAt:  jobStateModel.CreatedAt.Add(time.Duration(job.Spec.Timeout)),
	}, nil
}

func (s *Store) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
	inProgress, err := s.getLatestJobStatesWithStates(ctx, model.JobStateNew, model.JobStateQueued, model.JobStateInProgress)
	if err != nil {
		return nil, err
	}

	var out []model.JobWithInfo
	for _, p := range inProgress {
		j, err := s.GetJob(ctx, p.JobID)
		if err != nil {
			return nil, err
		}
		state, err := s.GetJobState(ctx, p.JobID)
		if err != nil {
			return nil, err
		}
		h, err := s.GetJobHistory(ctx, p.JobID, jobstore.JobHistoryFilterOptions{
			Since:                 time.Unix(0, 0).Unix(), // doubt there is anything prior to January 1st, 1970 at 00:00:00 UTC
			ExcludeExecutionLevel: false,
			ExcludeJobLevel:       false,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, model.JobWithInfo{
			Job:     j,
			State:   state,
			History: h,
		})
	}
	// sorts the slice such that earlier times come before later times.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Job.Metadata.CreatedAt.Before(out[j].Job.Metadata.CreatedAt)
	})
	return out, nil
}

func (s *Store) GetJobHistory(ctx context.Context, jobID string, options jobstore.JobHistoryFilterOptions) ([]model.JobHistory, error) {
	if options.ExcludeJobLevel && options.ExcludeExecutionLevel {
		return nil, nil
	}
	jobStates, err := s.getJobStateModels(ctx, jobID)
	if err != nil {
		return nil, err
	}
	executionStates, err := s.getExecutionStateModels(ctx, jobID)
	if err != nil {
		return nil, err
	}
	var out []model.JobHistory
	if !options.ExcludeJobLevel {
		for _, js := range jobStates {
			if js.CreatedAt.Before(time.Unix(options.Since, 0)) {
				continue
			}
			out = append(out, model.JobHistory{
				Type:  model.JobHistoryTypeJobLevel,
				JobID: jobID,
				JobState: &model.StateChange[model.JobStateType]{
					Previous: model.JobStateType(js.PreviousState),
					New:      model.JobStateType(js.CurrentState),
				},
				NewVersion: js.Version,
				Comment:    js.Comment,
				Time:       js.CreatedAt,
			})
		}
	}

	if !options.ExcludeExecutionLevel {
		for _, es := range executionStates {
			if es.CreatedAt.Before(time.Unix(options.Since, 0)) {
				continue
			}
			out = append(out, model.JobHistory{
				Type:             model.JobHistoryTypeExecutionLevel,
				JobID:            jobID,
				NodeID:           es.NodeID,
				ComputeReference: es.ComputeReference,
				ExecutionState: &model.StateChange[model.ExecutionStateType]{
					Previous: model.ExecutionStateType(es.PreviousState),
					New:      model.ExecutionStateType(es.CurrentState),
				},
				NewVersion: es.Version,
				Comment:    es.Comment,
				Time:       es.CreatedAt,
			})
		}
	}
	// sorts the slice such that earlier times come before later times
	// expectation is list is ordered by the time at which these states occurred as opposed to their versions.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Time.Before(out[j].Time)
	})
	return out, nil
}

func (s *Store) GetJobsCount(ctx context.Context, query jobstore.JobQuery) (int, error) {
	userQuery := query
	userQuery.Limit = 0
	userQuery.Offset = 0
	jobs, err := s.GetJobs(ctx, userQuery)
	if err != nil {
		return 0, err
	}
	return len(jobs), nil
}

func (s *Store) CreateExecution(ctx context.Context, executionID string, execution model.ExecutionState) error {
	if has, err := s.hasJob(ctx, execution.JobID); err != nil {
		return err
	} else if !has {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	if has, err := s.hasExecution(ctx, model.ExecutionID{
		JobID:       execution.JobID,
		NodeID:      execution.NodeID,
		ExecutionID: executionID,
	}); err != nil {
		return err
	} else if has {
		return jobstore.NewErrExecutionAlreadyExists(model.ExecutionID{
			JobID:       execution.JobID,
			NodeID:      execution.NodeID,
			ExecutionID: executionID,
		})
	}

	exeJson, err := json.Marshal(execution)
	if err != nil {
		return err
	}

	return s.Db.Create(&ExecutionState{
		JobID:            execution.JobID,
		NodeID:           execution.NodeID,
		ComputeReference: executionID,
		Version:          1,
		CurrentState:     int(execution.State),
		PreviousState:    int(execution.State),
		Comment:          "Execution Created",
		CreatedAt:        s.clock.Now(),
		Execution: pgtype.JSONB{
			Bytes:  exeJson,
			Status: pgtype.Present,
		},
	}).Error
}

func (s *Store) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	if has, err := s.hasJob(ctx, request.ExecutionID.JobID); err != nil {
		return err
	} else if !has {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}

	latestExecution, err := s.getLatestExecution(ctx, request.ExecutionID)
	if err != nil {
		return err
	}

	if err := request.Condition.Validate(latestExecution); err != nil {
		return err
	}

	if latestExecution.State.IsTerminal() {
		return jobstore.NewErrExecutionAlreadyTerminal(request.ExecutionID, latestExecution.State, request.NewValues.State)
	}

	eb, err := json.Marshal(request.NewValues)
	if err != nil {
		return err
	}

	return s.Db.WithContext(ctx).Create(&ExecutionState{
		JobID:            request.ExecutionID.JobID,
		NodeID:           request.ExecutionID.NodeID,
		ComputeReference: request.ExecutionID.ExecutionID,
		CurrentState:     int(request.NewValues.State),
		PreviousState:    int(latestExecution.State),
		Version:          latestExecution.Version + 1,
		Comment:          request.Comment,
		CreatedAt:        s.clock.Now(),
		Execution: pgtype.JSONB{
			Bytes:  eb,
			Status: pgtype.Present,
		},
	}).Error
}

//
// SQL Queries
//

func (s *Store) getLatestJobStatesWithStates(ctx context.Context, states ...model.JobStateType) ([]JobState, error) {
	queryJobState := make([]int, len(states))
	for i, state := range states {
		queryJobState[i] = int(state)
	}

	var result []JobState
	err := s.Db.WithContext(ctx).
		Table("job_states as js1").
		Select("js1.*").
		Joins("LEFT JOIN job_states as js2 ON js1.job_id = js2.job_id AND js1.version < js2.version").
		Where("js2.job_id IS NULL").
		Where("js1.current_state IN (?)", queryJobState).
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) getAllJobs(ctx context.Context) ([]model.Job, error) {
	var jobModels []Job
	if err := s.Db.WithContext(ctx).Find(&jobModels).Error; err != nil {
		return nil, err
	}
	var jobs []model.Job
	for _, j := range jobModels {
		job, err := j.AsJob()
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Store) getExecutionStateModels(ctx context.Context, id string) ([]ExecutionState, error) {
	var executionStates []ExecutionState
	if err := s.Db.WithContext(ctx).
		Where("job_id = ?", id).
		Find(&executionStates).Error; err != nil {
		// there may be no executions, and that is fine.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return executionStates, nil
}

func (s *Store) getJobStateModels(ctx context.Context, id string) ([]JobState, error) {
	// TODO remove when https://github.com/bacalhau-project/bacalhau/issues/2298 lands
	if len(id) < model.ShortIDLength {
		return nil, jobstore.NewErrJobNotFound(id)
	}
	if jobutils.ShortID(id) == id {
		return nil, fmt.Errorf("short JobIDs are not supported")
	}
	var jobStates []JobState
	if err := s.Db.WithContext(ctx).
		Where("job_id = ?", id).
		Find(&jobStates).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, jobstore.NewErrJobNotFound(id)
	}
	return jobStates, nil
}

func (s *Store) getLatestJobStateModel(ctx context.Context, id string) (*JobState, error) {
	// TODO remove when https://github.com/bacalhau-project/bacalhau/issues/2298 lands
	if len(id) < model.ShortIDLength {
		return nil, jobstore.NewErrJobNotFound(id)
	}
	if jobutils.ShortID(id) == id {
		return nil, fmt.Errorf("short JobIDs are not supported")
	}
	jobStateModel := new(JobState)
	res := s.Db.WithContext(ctx).
		Order("version desc").
		Limit(1).
		Find(jobStateModel, "job_id = ?", id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, jobstore.NewErrJobNotFound(id)
	}
	return jobStateModel, nil
}

func (s *Store) hasJob(ctx context.Context, id string) (bool, error) {
	// TODO remove when https://github.com/bacalhau-project/bacalhau/issues/2298 lands
	if len(id) < model.ShortIDLength {
		return false, jobstore.NewErrJobNotFound(id)
	}
	if jobutils.ShortID(id) == id {
		return false, fmt.Errorf("short JobIDs are not supported")
	}
	var exists bool
	if err := s.Db.WithContext(ctx).
		Model(&Job{}).
		Select("count(*) > 0").
		Where("job_id = ?", id).
		Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Store) hasExecution(ctx context.Context, id model.ExecutionID) (bool, error) {
	var exists bool
	if err := s.Db.WithContext(ctx).
		Model(&ExecutionState{}).
		Select("count(*) > 0").
		Where("job_id = ? and node_id = ? and compute_reference = ?", id.JobID, id.NodeID, id.ExecutionID).
		Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

// getLatestExecution fetches the latest execution state for a specific job_id, node_id, and compute_reference.
func (s *Store) getLatestExecution(ctx context.Context, id model.ExecutionID) (model.ExecutionState, error) {
	var latestExecution ExecutionState
	query := s.Db.Table("execution_states AS es").Select("es.*")
	query = query.Joins("JOIN (SELECT job_id, node_id, MAX(version) AS latest_version "+
		"FROM execution_states "+
		"WHERE job_id = ? AND node_id = ? AND compute_reference = ? "+
		"GROUP BY job_id, node_id) AS mv "+
		"ON es.job_id = mv.job_id AND es.node_id = mv.node_id AND es.version = mv.latest_version",
		id.JobID, id.NodeID, id.ExecutionID)

	res := query.WithContext(ctx).First(&latestExecution)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return model.ExecutionState{}, res.Error
		}
		return model.ExecutionState{}, jobstore.NewErrExecutionNotFound(id)
	}
	return latestExecution.AsExecutionState()
}

// getLatestExecutions fetches the latest execution states for each node_id associated with a given job_id from the execution_states table.
func (s *Store) getLatestExecutions(ctx context.Context, jobID string) ([]model.ExecutionState, error) {
	var latestExecutionStates []ExecutionState
	query := s.Db.Table("execution_states AS es").Select("es.*")
	query = query.Joins("JOIN "+
		"(SELECT node_id, MAX(version) AS latest_version "+
		"FROM execution_states "+
		"WHERE job_id = ? "+
		"GROUP BY node_id) "+
		"AS mv ON es.job_id = ? AND es.node_id = mv.node_id AND es.version = mv.latest_version",
		jobID, jobID)

	res := query.WithContext(ctx).Find(&latestExecutionStates)
	if res.Error != nil {
		return nil, res.Error
	}

	var executions []model.ExecutionState
	for _, e := range latestExecutionStates {
		state, err := e.AsExecutionState()
		if err != nil {
			return nil, err
		}
		executions = append(executions, state)
	}
	// sorts the slice such that earlier times come before later times.
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].CreateTime.Before(executions[j].CreateTime)
	})
	return executions, nil
}

func (s *Store) getJobWithShortID(ctx context.Context, id string) (model.Job, error) {
	if len(id) < model.ShortIDLength {
		return model.Job{}, bacerrors.NewJobNotFound(id)
	}

	var jobModel Job
	// support for short job IDs
	if jobutils.ShortID(id) == id {
		// passed in a short id, need to resolve the long id first
		if err := s.Db.Where(fmt.Sprintf("job_id LIKE '%%%s%%'", id)).Find(&jobModel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.Job{}, jobstore.NewErrJobNotFound(id)
			}
			return model.Job{}, err
		}
		return jobModel.AsJob()
	}
	res := s.Db.Limit(1).Find(&jobModel, "job_id = ?", id)
	if res.Error != nil {
		return model.Job{}, res.Error
	}
	if res.RowsAffected == 0 {
		return model.Job{}, jobstore.NewErrJobNotFound(id)
	}
	return jobModel.AsJob()
}
