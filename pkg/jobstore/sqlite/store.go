package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/imdario/mergo"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

func New(opts ...Option) (*Store, error) {
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

	if err := db.AutoMigrate(
		&Job{},
		&ShortIDMapping{},
		&JobState{},
		&Task{},
		&SpecConfig{},
		&InputSource{},
		&ResultPath{},
		&ResourceConfig{},
		&NetworkConfig{},
		&TimeoutConfig{},
		&Execution{},
		&ExecutionState{},
		&RunCommandResult{},
		&Evaluation{},
	); err != nil {
		return nil, err
	}
	return &Store{DB: db, clock: cfg.Clock}, nil
}

type Store struct {
	DB    *gorm.DB
	clock clock.Clock
}

//
// Job Operations
//

func (s *Store) GetJob(ctx context.Context, id string) (models.Job, error) {
	id, err := s.resolveJobID(ctx, id)
	if err != nil {
		return models.Job{}, err
	}
	if has, err := s.hasJob(ctx, id); err != nil {
		return models.Job{}, err
	} else if !has {
		return models.Job{}, bacerrors.NewJobNotFound(id)
	}

	j, err := s.getJob(ctx, id)
	if err != nil {
		return models.Job{}, err
	}

	state, err := s.getLatestJobState(ctx, id)
	if err != nil {
		return models.Job{}, err
	}

	return JobAndStateToDt(*j, *state)
}

func (s *Store) resolveJobID(ctx context.Context, id string) (string, error) {
	// TODO we should do this but can't because our tests expect different behavior..
	/*
		if len(id) < idgen.ShortIDLength {
			return "", fmt.Errorf("invalid jobID: %s", id)
		}
	*/
	if idgen.ShortID(id) != id {
		return id, nil
	}
	var mapping ShortIDMapping

	// Find the mapping by shortID
	result := s.DB.WithContext(ctx).Where("short_id = ?", id).First(&mapping)
	if result.Error != nil {
		return "", bacerrors.NewJobNotFound(id)
	}

	return mapping.JobID, nil
}

func (s *Store) getJob(ctx context.Context, id string) (*Job, error) {
	var out Job
	if err := s.DB.WithContext(ctx).
		Preload("Tasks").
		Model(&Job{}).
		Where("job_id = ?", id).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) hasJob(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := s.DB.Debug().WithContext(ctx).
		Model(&Job{}).
		Select("count(*) > 0").
		Where("job_id = ?", id).
		Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Store) CreateJob(ctx context.Context, j models.Job) error {
	// do we already have a record for this job?
	if has, err := s.hasJob(ctx, j.ID); err != nil {
		return err
	} else if has {
		return jobstore.NewErrJobAlreadyExists(j.ID)
	}

	// set unset fields and normalize to avoid panics
	// this poor little job store, so many responsibilities... why doesn't the caller do this?!
	now := s.clock.Now().UTC().UnixNano()
	j.State = models.NewJobState(models.JobStateTypePending)
	j.Revision = 1
	j.CreateTime = now
	// REALLY? I suppose we have modified it, but.. like.. nevermind
	j.ModifyTime = now
	// TODO: how about we don't hand poorly constructed stuff to the job store.
	j.Normalize()
	// really weird to reject a write due to invalid user args.
	if err := j.Validate(); err != nil {
		return err
	}

	// do the thing!
	return s.createJob(ctx, j)
}

func (s *Store) createJob(ctx context.Context, j models.Job) error {
	constraints, err := json.Marshal(j.Constraints)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(j.Meta)
	if err != nil {
		return err
	}
	labels, err := json.Marshal(j.Labels)
	if err != nil {
		return err
	}
	tasksModels, err := TasksToDt(j.ID, j.Tasks...)
	if err != nil {
		return err
	}
	jobModel := Job{
		JobID:       j.ID,
		Name:        j.Name,
		Namespace:   j.Namespace,
		Type:        j.Type,
		Priority:    j.Priority,
		Count:       j.Count,
		Constraints: constraints,
		Meta:        meta,
		Labels:      labels,
		CreatedTime: j.CreateTime,
		State: []JobState{{
			JobID:        j.ID,
			State:        int(j.State.StateType),
			Message:      j.State.Message,
			CreatedTime:  j.CreateTime,
			ModifiedTime: j.ModifyTime,
			Revision:     j.Revision,
			Version:      j.Version,
		}},
		Tasks: tasksModels,
	}
	IDMap := ShortIDMapping{
		JobID:   j.ID,
		ShortID: idgen.ShortID(j.ID),
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Create(&jobModel).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Create(&IDMap).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Store) UpdateJobState(ctx context.Context, request jobstore.UpdateJobStateRequest) error {
	// get the latest state for this job
	curState, err := s.getLatestJobState(ctx, request.JobID)
	if err != nil {
		return err
	}

	// cheating a bit as Validate and IsTerminal only need part of the Job, they are
	// assertions on its state.
	halfJob := models.Job{
		ID: curState.JobID,
		State: models.State[models.JobStateType]{
			StateType: models.JobStateType(curState.State),
			Message:   curState.Message,
		},
		Revision: curState.Revision,
	}
	if err := request.Condition.Validate(halfJob); err != nil {
		return err
	}

	// should we do this before validating??
	if halfJob.IsTerminal() {
		return jobstore.NewErrJobAlreadyTerminal(request.JobID, models.JobStateType(curState.State), request.NewState)
	}

	return s.updateJobState(ctx, curState, request)

}

func (s *Store) updateJobState(ctx context.Context, curState *JobState, request jobstore.UpdateJobStateRequest) error {
	return s.DB.WithContext(ctx).Create(&JobState{
		JobID:        request.JobID,
		State:        int(request.NewState),
		Message:      request.Comment,
		Version:      curState.Version + 1,
		Revision:     curState.Revision + 1,
		CreatedTime:  curState.CreatedTime, // TODO probably can drop this, its redundant by the JobModel
		ModifiedTime: s.clock.Now().UTC().UnixNano(),
	}).Error
}

func (s *Store) getLatestJobState(ctx context.Context, id string) (*JobState, error) {
	var state JobState

	if err := s.DB.WithContext(ctx).
		Where("job_id = ?", id).
		Order("id DESC"). // Assuming 'id' is an auto-incrementing primary key / a monotonic increasing index
		First(&state).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

// TODO test the hell out of this method.
func (s *Store) GetJobs(ctx context.Context, query jobstore.JobQuery) (*jobstore.JobQueryResponse, error) {
	var filtered []models.Job
	{
		// get all jobs in the name space and manually check requirements
		var allJobs []Job
		db := s.DB.Debug().WithContext(ctx)

		// Apply namespace filter if specified
		if query.Namespace != "" {
			db = db.Where("namespace = ?", query.Namespace)
		}

		if err := db.Find(&allJobs).Error; err != nil {
			return nil, err
		}

		// For each job, find the latest state
		result := make([]models.Job, len(allJobs))
		for i, job := range allJobs {
			state, err := s.getLatestJobState(ctx, job.JobID)
			if err != nil {
				return nil, err
			}
			result[i], err = JobAndStateToDt(job, *state)
			if err != nil {
				return nil, err
			}
		}

		filtered = make([]models.Job, 0, len(allJobs))

		for _, job := range result {
			// Initialize checks
			selectorMatch := query.Selector == nil || query.Selector.Matches(labels.Set(job.Labels))
			includeTagsCheck := len(query.IncludeTags) == 0 // Assume true if no include tags
			excludeTagsCheck := true                        // Assume true if no exclude tags

			// When ReturnAll is true, only exclude tags check is relevant
			if query.ReturnAll {
				selectorMatch = true    // Ignore selector match when ReturnAll is true
				includeTagsCheck = true // Ignore include tags check when ReturnAll is true
			} else {
				// Check for include tags match if include tags are provided
				if len(query.IncludeTags) > 0 {
					includeTagsCheck = false // Set to false initially, and only set to true if a match is found
					for _, tag := range query.IncludeTags {
						if _, ok := job.Labels[tag]; ok {
							includeTagsCheck = true
							break // Found a matching include tag, no need to check further
						}
					}
				}
			}

			// Check for exclude tags match if exclude tags are provided
			if len(query.ExcludeTags) > 0 {
				for _, tag := range query.ExcludeTags {
					if _, ok := job.Labels[tag]; ok {
						excludeTagsCheck = false // Found a matching exclude tag, this job should be excluded
						break                    // No need to check further exclude tags for this job
					}
				}
			}

			// Add the job to the filtered list only if it matches the selector and includes/excludes tags criteria
			if selectorMatch && includeTagsCheck && excludeTagsCheck {
				filtered = append(filtered, job)
			}
		}
	}

	jobIDs := make([]string, len(filtered))
	for i, f := range filtered {
		jobIDs[i] = f.ID
	}
	var jobs []Job

	db := s.DB.Preload("Tasks").Where("job_id IN ?", jobIDs).Where("deleted_at IS NULL")

	// Handle sorting
	if query.SortBy != "" {
		sortOrder := "ASC"
		if query.SortReverse {
			sortOrder = "DESC"
		}
		db = db.Order(fmt.Sprintf("%s %s", query.SortBy, sortOrder))
	} else {
		sortBy := "created_at"
		sortOrder := "ASC"
		if query.SortReverse {
			sortOrder = "DESC"
		}
		db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
	}

	// Handle pagination
	if !query.ReturnAll {
		if query.Limit != 0 {
			db = db.Offset(int(query.Offset)).Limit(int(query.Limit))
		} else {
			db = db.Offset(int(query.Offset))
		}
	} else {
		if query.Limit != 0 {
			db = db.Limit(int(query.Limit))
		}
		if query.Offset != 0 {
			db = db.Offset(int(query.Offset))
		}
	}

	// Execute query for jobs
	if err := db.Find(&jobs).Error; err != nil {
		return nil, err
	}

	// For each job, find the latest state
	result := make([]models.Job, len(jobs))
	for i, job := range jobs {
		state, err := s.getLatestJobState(ctx, job.JobID)
		if err != nil {
			return nil, err
		}
		result[i], err = JobAndStateToDt(job, *state)
		if err != nil {
			return nil, err
		}
	}

	// Prepare response
	response := jobstore.JobQueryResponse{
		Jobs:       result,
		Offset:     query.Offset,
		Limit:      query.Limit,
		NextOffset: query.Offset + uint32(len(jobs)), // Calculate next offset
	}

	// If the number of returned jobs is less than the limit, there are no more results
	if uint32(len(jobs)) < query.Limit {
		response.NextOffset = 0
	}

	return &response, nil
}

func (s *Store) GetInProgressJobs(ctx context.Context) ([]models.Job, error) {
	var inProgress []JobState
	excludedStates := []int{
		int(models.JobStateTypeCompleted),
		int(models.JobStateTypeFailed),
		int(models.JobStateTypeStopped),
	}

	err := s.DB.Raw(`
SELECT js.*
FROM job_states js
         INNER JOIN (
    SELECT job_id, MAX(id) AS MaxID
    FROM job_states
    GROUP BY job_id
) latest_js ON js.job_id = latest_js.job_id AND js.id = latest_js.MaxID
WHERE js.State NOT IN ?
`, excludedStates).Scan(&inProgress).Error
	if err != nil {
		return nil, err
	}

	out := make([]models.Job, len(inProgress))
	for i, js := range inProgress {
		j, err := s.getJob(ctx, js.JobID)
		if err != nil {
			return nil, err
		}
		out[i], err = JobAndStateToDt(*j, js)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) GetJobHistory(ctx context.Context, id string, options jobstore.JobHistoryFilterOptions) ([]models.JobHistory, error) {
	id, err := s.resolveJobID(ctx, id)
	if err != nil {
		return nil, err
	}
	since := time.Unix(options.Since, 0)

	var out []models.JobHistory
	if !options.ExcludeExecutionLevel {
		executionStates, err := s.getExecutionWithStatesForJobSince(ctx, id, since)
		if err != nil {
			return nil, err
		}

		prevState := models.ExecutionStateUndefined
		for _, e := range executionStates {
			out = append(out, models.JobHistory{
				Type:        models.JobHistoryTypeExecutionLevel,
				JobID:       id,
				NodeID:      e.NodeID,
				ExecutionID: e.ExecutionID,
				ExecutionState: &models.StateChange[models.ExecutionStateType]{
					Previous: prevState,
					New:      models.ExecutionStateType(e.ComputeState.State),
				},
				NewRevision: e.Revision,
				Comment:     e.ComputeState.Message,
				Time:        time.Unix(0, e.ModifiedTime),
			})
			prevState = models.ExecutionStateType(e.ComputeState.State)
		}
	}

	if !options.ExcludeJobLevel {
		jobStates, err := s.getJobStatesSince(ctx, id, since)
		if err != nil {
			return nil, err
		}
		prevState := models.JobStateTypeUndefined
		for _, j := range jobStates {
			out = append(out, models.JobHistory{
				Type:  models.JobHistoryTypeJobLevel,
				JobID: id,
				JobState: &models.StateChange[models.JobStateType]{
					Previous: prevState,
					New:      models.JobStateType(j.State),
				},
				Comment: j.Message,
				Time:    time.Unix(0, j.ModifiedTime),
			})
			prevState = models.JobStateType(j.State) // Update prevState with the current state for the next iteration
		}
	}

	// Filter out anything before the specified Since time, and anything that doesn't match the
	// specified ExecutionID or NodeID
	// TODO (tired forrest) copied this from the boltdb I think its the wrong filter
	out = lo.Filter(out, func(event models.JobHistory, index int) bool {
		if options.ExecutionID != "" && !strings.HasPrefix(event.ExecutionID, options.ExecutionID) {
			return false
		}

		if options.NodeID != "" && !strings.HasPrefix(event.NodeID, options.NodeID) {
			return false
		}

		if event.Time.Unix() < options.Since {
			return false
		}
		return true
	})

	sort.Slice(out, func(i, j int) bool {
		return out[i].Time.UTC().Before(out[j].Time.UTC())
	})
	return out, nil
}

func (s *Store) getJobStatesSince(ctx context.Context, id string, since time.Time) ([]JobState, error) {
	var states []JobState

	err := s.DB.WithContext(ctx).
		Model(&JobState{}).
		Where("job_id = ? AND created_at > ?", id, since).
		// TODO can use PK?
		Order("created_at DESC").
		Find(&states).Error

	if err != nil {
		return nil, err
	}

	return states, nil
}

func (s *Store) getExecutionWithStatesForJobSince(ctx context.Context, jobID string, since time.Time) ([]Execution, error) {
	var executions []Execution

	err := s.DB.WithContext(ctx).
		Model(&Execution{}).
		Where("job_id = ? AND created_at > ?", jobID, since). // Filter executions by job_id and time
		Order("created_at DESC").                             // Order by creation time
		Find(&executions).Error                               // Find all matching executions

	if err != nil {
		return nil, err
	}

	return executions, nil
}

//
// Execution Operations
//

func (s *Store) CreateExecution(ctx context.Context, e models.Execution) error {
	// TODO: don't hand half made stuff to jobstore, plz
	e.Normalize()

	resources, err := json.Marshal(e.AllocatedResources)
	if err != nil {
		return err
	}
	publishedResults, err := SpecConfigToDt(e.PublishedResult)
	if err != nil {
		return err
	}
	model := Execution{
		ExecutionID:        e.ID,
		EvaluationID:       e.EvalID,
		NodeID:             e.NodeID,
		JobID:              e.JobID,
		Namespace:          e.Namespace,
		Name:               e.Name,
		AllocatedResources: datatypes.JSON(resources),
		DesiredState: ExecutionState{
			ExecutionID: e.ID,
			State:       int(e.DesiredState.StateType),
			Message:     e.DesiredState.Message,
		},
		ComputeState: ExecutionState{
			ExecutionID: e.ID,
			State:       int(e.ComputeState.StateType),
			Message:     e.ComputeState.Message,
		},
		PublishedResult:   publishedResults,
		PreviousExecution: e.PreviousExecution,
		NextExecution:     e.NextExecution,
		FollowupEvalID:    e.FollowupEvalID,
		Revision:          e.Revision,
		CreateTime:        e.CreateTime,
		ModifiedTime:      e.ModifyTime,
	}
	if e.RunOutput != nil {
		model.RunOutput = RunCommandResult{
			STDOUT:          e.RunOutput.STDOUT,
			StdoutTruncated: e.RunOutput.StdoutTruncated,
			STDERR:          e.RunOutput.STDERR,
			StderrTruncated: e.RunOutput.StderrTruncated,
			ExitCode:        e.RunOutput.ExitCode,
			ErrorMsg:        e.RunOutput.ErrorMsg,
		}
	}
	now := s.clock.Now().UTC().UnixNano()
	if e.CreateTime == 0 {
		model.CreateTime = now
	}
	if e.ModifyTime == 0 {
		model.ModifiedTime = now
	}
	if e.Revision == 0 {
		model.Revision = 1
	}

	return s.DB.WithContext(ctx).Create(&model).Error
}

func (s *Store) hasExecution(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := s.DB.WithContext(ctx).
		Model(&Execution{}).
		Select("count(*) > 0").
		Where("execution_id = ?", id).
		Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Store) GetExecutions(ctx context.Context, query jobstore.GetExecutionsOptions) ([]models.Execution, error) {
	if has, err := s.hasJob(ctx, query.JobID); err != nil {
		return nil, err
	} else if !has {
		return nil, bacerrors.NewJobNotFound(query.JobID)
	}

	var executions []Execution
	err := s.DB.Raw(`
SELECT e.*
FROM executions e
         INNER JOIN (
    SELECT execution_id, MAX(revision) AS MaxRevision
    FROM executions
    WHERE job_id = ?
    GROUP BY execution_id
) AS latest_executions ON e.execution_id = latest_executions.execution_id
    AND e.revision = latest_executions.MaxRevision
WHERE e.job_id = ?
`, query.JobID, query.JobID).Scan(&executions).Error
	if err != nil {
		return nil, err
	}

	var j models.Job
	if query.IncludeJob {
		j, err = s.GetJob(ctx, query.JobID)
		if err != nil {
			return nil, err
		}
	}

	out := make([]models.Execution, len(executions))
	for i, e := range executions {
		out[i] = e.AsExecution()
		// TODO this is only used in testing and we are doing to remove the job filed from execution.
		if query.IncludeJob {
			out[i].Job = &j
		}
	}
	return out, nil
}

func (s *Store) getLatestExecution(ctx context.Context, executionID string) (*Execution, error) {
	var execution Execution

	err := s.DB.WithContext(ctx).
		Where("execution_id = ?", executionID).
		Order("id DESC").       // Assuming 'id' is an auto-incrementing primary key or a monotonic increasing index
		First(&execution).Error // Get the latest execution based on ID

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func (s *Store) UpdateExecution(ctx context.Context, request jobstore.UpdateExecutionRequest) error {
	curExecution, err := s.getLatestExecution(ctx, request.ExecutionID)
	if err != nil {
		return err
	}

	// TODO this state shouldn't be leaking into the job store, why does the store care? do this higher up.
	asExecution := curExecution.AsExecution()
	if err := request.Condition.Validate(asExecution); err != nil {
		return err
	}
	// TODO this state shouldn't be leaking into the job store, why does the store care? do this higher up.
	if asExecution.IsTerminalComputeState() {
		return jobstore.NewErrExecutionAlreadyTerminal(
			request.ExecutionID, asExecution.ComputeState.StateType, request.NewValues.ComputeState.StateType)
	}

	newExecution := request.NewValues
	newExecution.CreateTime = curExecution.CreateTime
	if newExecution.ModifyTime == 0 {
		newExecution.ModifyTime = s.clock.Now().UTC().UnixNano()
	}
	if newExecution.Revision == 0 {
		newExecution.Revision = curExecution.Revision + 1
	}
	newExecution.Normalize()

	if err := mergo.Merge(&newExecution, curExecution.AsExecution()); err != nil {
		return err
	}

	return s.CreateExecution(ctx, newExecution)
}

//
// Evaluation Operation
//

func (s *Store) CreateEvaluation(ctx context.Context, eval models.Evaluation) error {
	if has, err := s.hasJob(ctx, eval.JobID); err != nil {
		return err
	} else if !has {
		return jobstore.NewErrJobNotFound(eval.ID)
	}
	if has, err := s.hasEvaluation(ctx, eval.ID); err != nil {
		return err
	} else if has {
		// TODO this error shouldn't be here?? wrong type or too many types
		return bacerrors.NewAlreadyExists(eval.ID, "Evaluation")
	}

	model := Evaluation{
		EvaluationID: eval.ID,
		JobID:        eval.JobID,
		TriggeredBy:  eval.TriggeredBy,
		Priority:     eval.Priority,
		Type:         eval.Type,
		Status:       eval.Status,
		Comment:      eval.Comment,
		WaitUntil:    eval.WaitUntil,
		CreatedTime:  eval.CreateTime,
		ModifiedTime: eval.ModifyTime,
	}

	return s.DB.Create(&model).Error
}

func (s *Store) hasEvaluation(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := s.DB.WithContext(ctx).
		Model(&Evaluation{}).
		Select("count(*) > 0").
		Where("evaluation_id = ?", id).
		Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Store) GetEvaluation(ctx context.Context, id string) (models.Evaluation, error) {
	eval, err := s.getLatestEvaluation(ctx, id)
	if err != nil {
		return models.Evaluation{}, err
	}
	return eval.AsEvaluation(), nil
}

func (s *Store) getLatestEvaluation(ctx context.Context, id string) (*Evaluation, error) {
	var evaluation Evaluation

	err := s.DB.WithContext(ctx).
		Where("evaluation_id = ?", id).
		Order("id DESC"). // 'id' is an auto-incrementing primary key / monotonic increasing index
		First(&evaluation).Error

	if err != nil {
		return nil, err
	}

	return &evaluation, nil
}

func (s *Store) Close(ctx context.Context) error {
	d, err := s.DB.DB()
	if err != nil {
		return err
	}
	return d.Close()
}

//
// Unused method
//

func (s *Store) DeleteEvaluation(ctx context.Context, id string) error {
	return fmt.Errorf("no implement")
}

func (s *Store) DeleteJob(ctx context.Context, jobID string) error {
	return fmt.Errorf("no implement")
}

func (s *Store) Watch(ctx context.Context, types jobstore.StoreWatcherType, events jobstore.StoreEventType) chan jobstore.WatchEvent {
	panic("not implement and not called in code")
}
