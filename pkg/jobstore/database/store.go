package database

import (
	"context"
	"encoding/json"
	"errors"
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
	if err := db.AutoMigrate(&Job{}, &JobState{}, &ExecutionState{}); err != nil {
		return nil, err
	}
	return &DatabaseStore{Db: db}, nil
}

// Static check to ensure that Transport implements Transport:
var _ jobstore.Store = (*DatabaseStore)(nil)

type DatabaseStore struct {
	Db *gorm.DB
}

func (d *DatabaseStore) CreateExecutionBid(ctx context.Context, jobID string, nodeID string) error {
	var jobModel Job
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", jobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(jobID)
	}

	var bid ExecutionBid
	res = d.Db.Limit(1).Find(&bid, "job_id = ? and node_id = ?", jobID, nodeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 0 {
		return jobstore.NewErrExecutionAlreadyExists(model.ExecutionID{
			JobID:       jobID,
			NodeID:      nodeID,
			ExecutionID: "",
		})
	}

	return d.Db.Create(&ExecutionBid{
		JobID:     jobID,
		NodeID:    nodeID,
		CreatedAt: time.Now(),
		State:     int(model.ExecutionStateAskForBid),
	}).Error
}

func (d *DatabaseStore) UpdateExecutionBid(ctx context.Context, jobID, nodeID string, request jobstore.UpdateExecutionBidRequest) error {
	var jobModel Job
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", jobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(jobID)
	}

	var bid ExecutionBid
	res = d.Db.Limit(1).Find(&bid, "job_id = ? and node_id = ?", jobID, nodeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrExecutionNotFound(model.ExecutionID{
			JobID:       jobID,
			NodeID:      nodeID,
			ExecutionID: "",
		})
	}
	return d.Db.Transaction(func(tx *gorm.DB) error {
		// remove bid and create an ExecutionState from it. Then create a second execution state to represent the current state.
		// We delete it to allow the same node to re-bid later without an ErrAlreadyExists returned. I am not condident on this and
		// the fact that a complete "executionID" doesn't exist when the execution is created is very painful, as its missing the ComputeReference.
		if err := d.Db.Delete(bid).Error; err != nil {
			return err
		}
		if err := d.Db.Create(ExecutionState{
			JobID:            jobID,
			NodeID:           nodeID,
			ComputeReference: request.ComputeReference,
			CurrentState:     bid.State,
			PreviousState:    bid.State,
			Version:          1,
			CreatedAt:        bid.CreatedAt,
		}).Error; err != nil {
			return err
		}
		return d.Db.Create(ExecutionState{
			JobID:            jobID,
			NodeID:           nodeID,
			ComputeReference: request.ComputeReference,
			Comment:          request.Comment,
			CurrentState:     int(request.NewState),
			PreviousState:    bid.State,
			Version:          2,
			CreatedAt:        time.Now(),
		}).Error
	})
}

func (d *DatabaseStore) UpdateExecutionState(ctx context.Context, id model.ExecutionID, request jobstore.UpdateExecutionStateRequest) error {
	//TODO implement me
	panic("implement me")
}

func (d *DatabaseStore) UpdateExecutionOutputs(ctx context.Context, id model.ExecutionID, request jobstore.UpdateExecutionOutputRequest) error {
	//TODO implement me
	panic("implement me")
}

func (d *DatabaseStore) UpdateExecutionVerification(ctx context.Context, id model.ExecutionID, request jobstore.UpdateExecutionVerificationRequest) error {
	//TODO implement me
	panic("implement me")
}

func (d *DatabaseStore) ExecutionComplete(ctx context.Context, id model.ExecutionID, request jobstore.ExecutionCompleteRequest) error {
	//TODO implement me
	panic("implement me")
}

func (d *DatabaseStore) ExecutionFailed(ctx context.Context, id model.ExecutionID, request jobstore.ExecutionFailedRequest) error {
	//TODO implement me
	panic("implement me")
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

	var latestExecutionStates []ExecutionState
	query := d.Db.Table("execution_states AS es").Select("es.*")
	query = query.Joins("JOIN (SELECT node_id, MAX(version) AS latest_version FROM execution_states WHERE job_id = ? GROUP BY node_id) AS mv ON es.job_id = ? AND es.node_id = mv.node_id AND es.version = mv.latest_version", jobID, jobID)
	res = query.Find(&latestExecutionStates)
	if res.Error != nil {
		return model.JobState{}, res.Error
	}

	var executions []model.ExecutionState
	for _, e := range latestExecutionStates {
		var tmp model.ExecutionState
		if err := e.Execution.AssignTo(&tmp); err != nil {
			return model.JobState{}, err
		}
		tmp.Version = e.Version
		tmp.CreateTime = e.CreatedAt
		executions = append(executions, tmp)
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
			Type:  model.JobHistoryTypeExecutionLevel,
			JobID: jobID,
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
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", execution.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(execution.JobID)
	}

	var state ExecutionState
	res = d.Db.Limit(1).Find(&state, "job_id = ? and node_id = ?", execution.JobID, execution.NodeID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 0 {
		return jobstore.NewErrExecutionAlreadyExists(execution.ID())
	}

	eb, err := json.Marshal(execution)
	if err != nil {
		return err
	}
	return d.Db.Create(&ExecutionState{
		JobID:            execution.JobID,
		NodeID:           execution.NodeID,
		ComputeReference: execution.ComputeReference,
		CurrentState:     int(execution.State),
		PreviousState:    int(execution.State),
		Version:          1,
		CreatedAt:        time.Now(),
		Execution: pgtype.JSONB{
			Bytes:  eb,
			Status: pgtype.Present,
		},
	}).Error
}

func (d *DatabaseStore) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	var jobModel Job
	res := d.Db.Limit(1).Find(&jobModel, "job_id = ?", request.ExecutionID.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return jobstore.NewErrJobNotFound(request.ExecutionID.JobID)
	}

	// we found a job and bid for this execution, safe to say it has been created.
	// check if it has an existing state
	var latestExecution ExecutionState
	query := d.Db.Table("execution_states AS es").Select("es.*")
	query = query.Joins("JOIN (SELECT job_id, node_id, MAX(version) AS latest_version FROM execution_states WHERE job_id = ? AND node_id = ? GROUP BY job_id, node_id) AS mv ON es.job_id = mv.job_id AND es.node_id = mv.node_id AND es.version = mv.latest_version", request.ExecutionID.JobID, request.ExecutionID.NodeID)
	res = query.First(&latestExecution)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return res.Error
		}
		return jobstore.NewErrExecutionNotFound(request.ExecutionID)
	}
	if err := ValidateExecutionRequest(request, latestExecution); err != nil {
		return err
	}
	if model.ExecutionStateType(latestExecution.CurrentState).IsTerminal() {
		return jobstore.NewErrExecutionAlreadyTerminal(request.ExecutionID, model.ExecutionStateType(latestExecution.CurrentState), request.NewValues.State)
	}
	eb, err := json.Marshal(request.NewValues)
	if err != nil {
		return err
	}
	return d.Db.Create(&ExecutionState{
		JobID:            request.ExecutionID.JobID,
		NodeID:           request.ExecutionID.NodeID,
		ComputeReference: request.ExecutionID.ExecutionID,
		CurrentState:     int(request.NewValues.State),
		PreviousState:    latestExecution.CurrentState,
		Version:          latestExecution.Version + 1,
		CreatedAt:        time.Now(),
		Execution: pgtype.JSONB{
			Bytes:  eb,
			Status: pgtype.Present,
		},
	}).Error
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
