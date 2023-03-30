package database

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sort"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/jackc/pgtype"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func init() {
	logging.SetDebugLogging()
}

func NewDatabaseStore(dial gorm.Dialector) (*DatabaseStore, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)
	db, err := gorm.Open(dial, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 newLogger,
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Job{}, &JobState{}, &JobExecution{}, &NodeExecution{}, &ExecutionState{}, &ExecutionOutput{}, &ExecutionVerificationProposal{}, &ExecutionVerificationResult{}, &ExecutionPublishResult{}); err != nil {
		return nil, err
	}
	return &DatabaseStore{Db: db}, nil
}

type DatabaseStore struct {
	Db *gorm.DB
}

func (d *DatabaseStore) CreateJob(ctx context.Context, j model.Job) error {
	var jobModel Job
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", j.ID())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 0 {
		return jobstore.NewErrJobAlreadyExists(j.ID())
	}

	now := time.Now()
	jb, err := json.Marshal(j)
	if err != nil {
		return err
	}
	jobModel = Job{
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
		CreatedAt:     now,
	}
	return d.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&jobModel).Error; err != nil {
			return err
		}

		if err := tx.Create(&jobStateModel).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DatabaseStore) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) error {
	var jobStateModel JobState
	res := d.Db.Order("version desc").Limit(1).Find(&jobStateModel, "job_id = ?", request.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(request.JobID)
	}
	if err := ValidateJobRequest(request, jobStateModel); err != nil {
		return err
	}
	if model.JobStateType(jobStateModel.CurrentState).IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(jobStateModel.JobID, model.JobStateType(jobStateModel.CurrentState), request.NewState)
	}
	newJobStateModel := &JobState{
		JobID:         jobStateModel.JobID,
		Version:       jobStateModel.Version + 1,
		CurrentState:  int(request.NewState),
		PreviousState: jobStateModel.CurrentState,
		CreatedAt:     time.Now(),
	}
	return d.Db.Create(newJobStateModel).Error
}

func (d *DatabaseStore) GetJob(ctx context.Context, id string) (model.Job, error) {
	var jobModel Job
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", id)
	if res.Error != nil {
		return model.Job{}, res.Error
	}
	if res.RowsAffected == 0 {
		return model.Job{}, jobstore.NewErrJobNotFound(id)
	}
	var out model.Job
	if err := jobModel.Job.AssignTo(&out); err != nil {
		return model.Job{}, err
	}
	return out, nil
}

func (d *DatabaseStore) GetJobs(ctx context.Context, query jobstore.JobQuery) ([]model.Job, error) {
	if query.ID != "" {
		j, err := d.GetJob(ctx, query.ID)
		if err != nil {
			return nil, err
		}
		return []model.Job{j}, nil
	}
	// TODO write an actual query
	// - not doing this now since the Job Model contains model.Job as json and json queries are not consisten across Db impls.
	var jobModels []Job
	if err := d.Db.Find(&jobModels).Error; err != nil {
		return nil, err
	}
	var jobs []model.Job
	for _, j := range jobModels {
		cj := model.Job{}
		if err := j.Job.AssignTo(cj); err != nil {
			return nil, err
		}
		jobs = append(jobs, cj)
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

func (d *DatabaseStore) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	var jobStateModel JobState
	res := d.Db.Order("version desc").Limit(1).Find(&jobStateModel, "job_id = ?", jobID)
	if res.Error != nil {
		return model.JobState{}, res.Error
	}
	if res.RowsAffected == 0 {
		return model.JobState{}, jobstore.NewErrJobNotFound(jobID)
	}
	var executionStateModels []ExecutionState
	query := d.Db.Table("execution_states AS es").Select("es.job_id, es.node_id, es.compute_reference, es.execution_id, es.status, es.version, es.current_state, es.previous_state, es.created_at")
	query = query.Joins("JOIN (SELECT job_id, execution_id, MAX(version) AS latest_version FROM execution_states WHERE job_id = ? GROUP BY job_id, execution_id) AS mv ON es.job_id = mv.job_id AND es.execution_id = mv.execution_id AND es.version = mv.latest_version", jobID)
	query = query.Order("es.version DESC")
	if err := query.Find(&executionStateModels).Error; err != nil {
		return model.JobState{}, err
	}
	/*
		var executionStateModels []ExecutionState
		res = d.Db.Order("version desc").Where("job_id = ?", jobID).Find(&executionStateModels)
		if res.Error != nil {
			return model.JobState{}, res.Error
		}

	*/
	var executions []model.ExecutionState
	for _, exe := range executionStateModels {
		runOutput, err := d.getExecutionOutput(ctx, exe.ExecutionID, exe.Version)
		if err != nil {
			return model.JobState{}, err
		}
		proposal, err := d.getExecutionProposal(ctx, exe.ExecutionID, exe.Version)
		if err != nil {
			return model.JobState{}, err
		}
		result, err := d.getExecutionResult(ctx, exe.ExecutionID, exe.Version)
		if err != nil {
			return model.JobState{}, err
		}
		publish, err := d.getExecutionPublish(ctx, exe.ExecutionID, exe.Version)
		if err != nil {
			return model.JobState{}, err
		}
		executions = append(executions, model.ExecutionState{
			JobID:                exe.JobID,
			NodeID:               exe.NodeID,
			ComputeReference:     exe.ComputeReference,
			State:                model.ExecutionStateType(exe.CurrentState),
			Status:               exe.Status,
			VerificationProposal: proposal,
			VerificationResult:   result,
			PublishedResult:      publish,
			RunOutput:            runOutput,
			Version:              exe.Version,
			CreateTime:           exe.CreatedAt,
		})
	}
	job, err := d.GetJob(ctx, jobID)
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

func (d *DatabaseStore) getExecutionPublish(ctx context.Context, exeID string, version int) (model.StorageSpec, error) {
	var exeModel ExecutionPublishResult
	res := d.Db.Where("execution_id = ? AND version = ?", exeID, version).Find(&exeModel)
	if res.Error != nil {
		return model.StorageSpec{}, nil
	}
	if res.RowsAffected == 0 {
		return model.StorageSpec{}, nil
	}
	var out model.StorageSpec
	if err := exeModel.Result.AssignTo(&out); err != nil {
		return model.StorageSpec{}, err
	}
	return out, nil
}

func (d *DatabaseStore) getExecutionResult(ctx context.Context, exeID string, version int) (model.VerificationResult, error) {
	var exeModel ExecutionVerificationResult
	res := d.Db.Where("execution_id = ? AND version = ?", exeID, version).Find(&exeModel)
	if res.Error != nil {
		return model.VerificationResult{}, res.Error
	}
	if res.RowsAffected == 0 {
		return model.VerificationResult{}, nil
	}
	return model.VerificationResult{
		Complete: exeModel.Complete,
		Result:   exeModel.Result,
	}, nil
}

func (d *DatabaseStore) getExecutionOutput(ctx context.Context, exeID string, version int) (*model.RunCommandResult, error) {
	var exeOutputModel ExecutionOutput
	res := d.Db.Where("execution_id = ? AND version = ?", exeID, version).Find(&exeOutputModel)
	if res.Error != nil {
		return nil, res.Error
	}
	// TODO this means there isnt any output yet
	if res.RowsAffected == 0 {
		return nil, nil
	}
	out := new(model.RunCommandResult)
	if err := exeOutputModel.Output.AssignTo(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *DatabaseStore) getExecutionProposal(ctx context.Context, exeID string, version int) ([]byte, error) {
	var exeModel ExecutionVerificationProposal
	res := d.Db.Where("execution_id = ? AND version = ?", exeID, version).Find(&exeModel)
	if res.Error != nil {
		return nil, res.Error
	}
	// TODO this means there isnt any output yet
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return exeModel.Proposal, nil
}

func (d *DatabaseStore) GetInProgressJobs(ctx context.Context) ([]model.JobWithInfo, error) {
	var inProgress []JobState
	if err := d.Db.Not(map[string]interface{}{"current_state": []int{
		int(model.JobStateCompleted),
		int(model.JobStateError),
		int(model.JobStateCancelled),
		int(model.JobStateCompletedPartially),
	}}).Find(&inProgress).Error; err != nil {
		return nil, err
	}
	var out []model.JobWithInfo
	for _, p := range inProgress {
		j, err := d.GetJob(ctx, p.JobID)
		if err != nil {
			return nil, err
		}
		s, err := d.GetJobState(ctx, p.JobID)
		if err != nil {
			return nil, err
		}
		h, err := d.GetJobHistory(ctx, p.JobID, time.Unix(0, 0))
		if err != nil {
			return nil, err
		}
		out = append(out, model.JobWithInfo{
			Job:     j,
			State:   s,
			History: h,
		})
	}
	return out, nil
}

// TODO query with since
func (d *DatabaseStore) GetJobHistory(ctx context.Context, jobID string, since time.Time) ([]model.JobHistory, error) {
	var jobStates []JobState
	if err := d.Db.Where("job_id = ?", jobID).Find(&jobStates).Error; err != nil {
		return nil, err
	}
	var executionStates []ExecutionState
	if err := d.Db.Where("job_id = ?", jobID).Find(&executionStates).Error; err != nil {
		return nil, err
	}
	var out []model.JobHistory
	for _, js := range jobStates {
		out = append(out, model.JobHistory{
			Type:  model.JobHistoryTypeJobLevel,
			JobID: jobID,
			JobState: &model.StateChange[model.JobStateType]{
				Previous: model.JobStateType(js.PreviousState),
				New:      model.JobStateType(js.CurrentState),
			},
			NewVersion: js.Version,
			Comment:    "",
			Time:       js.CreatedAt,
		})
	}

	for _, es := range executionStates {
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
			Comment:    "",
			Time:       es.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		// TODO double check ordering
		// TODO consider ordering by version
		return out[i].Time.Before(out[j].Time)
	})
	return out, nil
}

func (d *DatabaseStore) GetJobsCount(ctx context.Context, query jobstore.JobQuery) (int, error) {
	userQuery := query
	userQuery.Limit = 0
	userQuery.Offset = 0
	jobs, err := d.GetJobs(ctx, userQuery)
	if err != nil {
		return 0, err
	}
	return len(jobs), nil
}

func (d *DatabaseStore) CreateExecution(ctx context.Context, execution model.ExecutionState) error {
	var jobModel Job
	res := d.Db.Find(&jobModel, "job_id = ?", execution.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	var exeStateModel ExecutionState
	res = d.Db.Find(&exeStateModel, "execution_id = ?", execution.ID().String())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 0 {
		return jobstore.NewErrExecutionAlreadyExists(execution.ID())
	}

	now := time.Now()
	return d.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ExecutionState{
			JobID:            execution.JobID,
			NodeID:           execution.NodeID,
			ComputeReference: execution.ComputeReference,
			Status:           execution.Status,
			Version:          1,
			CurrentState:     int(execution.State),
			PreviousState:    int(execution.State),
			CreatedAt:        now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&JobExecution{
			JobID:       execution.JobID,
			ExecutionID: execution.ID().String(),
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&NodeExecution{
			NodeID:      execution.NodeID,
			ExecutionID: execution.ID().String(),
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DatabaseStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	var jobModel Job
	// TODO order by
	res := d.Db.Find(&jobModel, "job_id = ?", request.ExecutionID.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}

	var exeStateModel ExecutionState
	// TODO order by
	res = d.Db.Find(&exeStateModel, "execution_id = ?", request.ExecutionID.String())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}

	if err := ValidateExecutionRequest(request, exeStateModel); err != nil {
		return err
	}

	if model.ExecutionStateType(exeStateModel.CurrentState).IsTerminal() {
		return jobstore.NewErrExecutionAlreadyTerminal(request.ExecutionID, model.ExecutionStateType(exeStateModel.CurrentState), request.NewValues.State)
	}

	now := time.Now()
	version := exeStateModel.Version + 1
	return d.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ExecutionState{
			JobID:  request.ExecutionID.JobID,
			NodeID: request.ExecutionID.NodeID,
			// TODO unsure if this is right
			ComputeReference: request.NewValues.ComputeReference,
			Status:           request.NewValues.Status,
			Version:          version,
			CurrentState:     int(request.NewValues.State),
			PreviousState:    exeStateModel.CurrentState,
			CreatedAt:        now,
		}).Error; err != nil {
			return err
		}
		if request.NewValues.RunOutput != nil {
			m, err := newExecutionOutputModel(request.ExecutionID, version, now, request.NewValues.RunOutput)
			if err != nil {
				return err
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}
		if len(request.NewValues.VerificationProposal) > 0 {
			if err := tx.Create(&ExecutionVerificationProposal{
				ExecutionID: request.ExecutionID.String(),
				Version:     version,
				Proposal:    request.NewValues.VerificationProposal,
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		if request.NewValues.State == model.ExecutionStateResultRejected ||
			request.NewValues.State == model.ExecutionStateResultAccepted {
			if err := tx.Create(&ExecutionVerificationResult{
				ExecutionID: request.ExecutionID.String(),
				Version:     version,
				Complete:    request.NewValues.VerificationResult.Complete,
				Result:      request.NewValues.VerificationResult.Result,
				CreatedAt:   now,
			}).Error; err != nil {
				return err
			}
		}
		if request.NewValues.State == model.ExecutionStateCompleted {
			m, err := newExecutionPublishResultMode(request.ExecutionID, version, now, request.NewValues.PublishedResult)
			if err != nil {
				return err
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func newExecutionOutputModel(executionID model.ExecutionID, version int, now time.Time, output *model.RunCommandResult) (*ExecutionOutput, error) {
	jr, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return &ExecutionOutput{
		ExecutionID: executionID.String(),
		Version:     version,
		Output: pgtype.JSONB{
			Bytes:  jr,
			Status: pgtype.Present,
		},
		CreatedAt: now,
	}, nil
}

func newExecutionPublishResultMode(executionID model.ExecutionID, version int, now time.Time, result model.StorageSpec) (*ExecutionPublishResult, error) {
	js, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &ExecutionPublishResult{
		ExecutionID: executionID.String(),
		Version:     version,
		Result: pgtype.JSONB{
			Bytes:  js,
			Status: pgtype.Present,
		},
		CreatedAt: now,
	}, nil
}

func ValidateJobRequest(request jobstore.UpdateJobStateRequest, state JobState) error {
	if request.Condition.ExpectedState != model.JobStateNew && request.Condition.ExpectedState != model.JobStateType(state.CurrentState) {
		return jobstore.NewErrInvalidJobState(state.JobID, model.JobStateType(state.CurrentState), request.Condition.ExpectedState)
	}
	if request.Condition.ExpectedVersion != 0 && request.Condition.ExpectedVersion != state.Version {
		return jobstore.NewErrInvalidJobVersion(state.JobID, state.Version, request.Condition.ExpectedVersion)
	}
	if len(request.Condition.UnexpectedStates) > 0 {
		for _, s := range request.Condition.UnexpectedStates {
			if s == model.JobStateType(state.CurrentState) {
				return jobstore.NewErrInvalidJobState(state.JobID, model.JobStateType(state.CurrentState), model.JobStateNew)
			}
		}
	}
	return nil
}

func ValidateExecutionRequest(request jobstore.UpdateExecutionRequest, execution ExecutionState) error {
	if request.Condition.ExpectedState != model.ExecutionStateNew && request.Condition.ExpectedState != model.ExecutionStateType(execution.CurrentState) {
		return jobstore.NewErrInvalidExecutionState(request.ExecutionID, model.ExecutionStateType(execution.CurrentState), request.Condition.ExpectedState)
	}
	if request.Condition.ExpectedVersion != 0 && request.Condition.ExpectedVersion != execution.Version {
		return jobstore.NewErrInvalidExecutionVersion(request.ExecutionID, execution.Version, request.Condition.ExpectedVersion)
	}
	if len(request.Condition.UnexpectedStates) > 0 {
		for _, s := range request.Condition.UnexpectedStates {
			if s == model.ExecutionStateType(execution.CurrentState) {
				return jobstore.NewErrInvalidExecutionState(request.ExecutionID, model.ExecutionStateType(execution.CurrentState), model.ExecutionStateNew)
			}
		}
	}
	return nil
}
