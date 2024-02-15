package sqlite

import (
	"encoding/json"
	"time"

	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func (j *Job) AsJob() models.Job {
	return models.Job{
		ID:        j.JobID,
		Name:      j.Name,
		Namespace: j.Namespace,
		Type:      j.Type,
		Priority:  j.Priority,
		Count:     j.Count,
		// Constraints: j.Constraints,
		// Meta:        j.Meta,
		// Labels:      j.Labels,
		// Tasks:       j.Tasks,
		State: models.State[models.JobStateType]{
			StateType: models.JobStateType(j.State.State),
			Message:   j.State.Message,
		},
		Version:    j.State.Version,
		Revision:   j.State.Revision,
		CreateTime: j.State.CreatedTime,
		ModifyTime: j.State.ModifiedTime,
	}
}

type Job struct {
	gorm.Model         // Includes fields ID, CreatedAt, UpdatedAt, DeletedAt
	JobID       string `gorm:"primaryKey"`
	Name        string
	Namespace   string
	Type        string
	Priority    int
	Count       int
	Constraints datatypes.JSON
	Meta        datatypes.JSON
	Labels      datatypes.JSON
	CreatedTime int64
	// Associations
	State JobState `gorm:"foreignKey:JobID;references:JobID"`
	Tasks []Task   `gorm:"foreignKey:JobID;references:JobID"` // Associating tasks with jobs
}

type JobState struct {
	gorm.Model
	JobID   string
	State   int
	Message string

	Version      uint64
	Revision     uint64
	CreatedTime  int64
	ModifiedTime int64
}

func ToJobStateModel(j models.Job) JobState {
	return JobState{
		JobID:   j.ID,
		State:   int(j.State.StateType),
		Message: j.State.Message,
	}
}

type Task struct {
	gorm.Model          // Includes fields ID, CreatedAt, UpdatedAt, DeletedAt
	JobID        string // Foreign key from Job
	Name         string
	Engine       SpecConfig `gorm:"embedded;embeddedPrefix:engine_"`
	Publisher    SpecConfig `gorm:"embedded;embeddedPrefix:publisher_"`
	Env          datatypes.JSON
	Meta         datatypes.JSON
	InputSources []InputSource  `gorm:"foreignKey:TaskID;references:ID"`
	ResultPaths  []ResultPath   `gorm:"foreignKey:TaskID;references:ID"`
	Resources    ResourceConfig `gorm:"embedded"`
	Network      NetworkConfig  `gorm:"embedded"`
	Timeouts     TimeoutConfig  `gorm:"embedded"`
}

func ToTaskModel(j models.Job) []Task {
	var tasks []Task
	for _, dt := range j.Tasks {
		// Convert each domain task to a GORM Task model
		// This includes converting any nested structures or fields as necessary
		task := Task{
			JobID:        j.ID,
			Name:         dt.Name,
			Engine:       ToSpecConfigModel(dt.Engine),
			Publisher:    ToSpecConfigModel(dt.Publisher),
			Env:          datatypes.JSON(lo.Must(json.Marshal(dt.Env))),
			Meta:         datatypes.JSON(lo.Must(json.Marshal(dt.Meta))),
			InputSources: ToInputSourceModel(dt),
			ResultPaths:  ToResultPathModel(dt),
			Resources: ResourceConfig{
				CPU:    dt.ResourcesConfig.CPU,
				Memory: dt.ResourcesConfig.Memory,
				Disk:   dt.ResourcesConfig.Disk,
				GPU:    dt.ResourcesConfig.CPU,
			},
			Network: NetworkConfig{
				Type:    dt.Network.Type.String(),
				Domains: lo.Must(json.Marshal(dt.Network.Domains)),
			},
			Timeouts: TimeoutConfig{ExecutionTimeout: dt.Timeouts.ExecutionTimeout},
		}
		tasks = append(tasks, task)
	}
	return tasks
}

type SpecConfig struct {
	Type   string
	Params datatypes.JSON // Using JSON data type, assuming Params is a JSON object
}

func ToSpecConfigModel(s *models.SpecConfig) SpecConfig {
	if s == nil {
		return SpecConfig{}
	}
	out := SpecConfig{
		Type: s.Type,
	}
	if s.Params != nil {
		pb := lo.Must(json.Marshal(s.Params))
		out.Params = pb
	}
	return out
}

type InputSource struct {
	gorm.Model      // Includes fields ID, CreatedAt, UpdatedAt, DeletedAt
	TaskID     uint // Foreign key from Task
	Alias      string
	Target     string
	Source     SpecConfig `gorm:"embedded;embeddedPrefix:source_"`
}

func ToInputSourceModel(t *models.Task) []InputSource {
	var inputs []InputSource
	for _, di := range t.InputSources {
		i := InputSource{
			Alias:  di.Alias,
			Target: di.Target,
			Source: ToSpecConfigModel(di.Source),
		}
		inputs = append(inputs, i)
	}
	return inputs
}

type ResultPath struct {
	gorm.Model      // Includes fields ID, CreatedAt, UpdatedAt, DeletedAt
	TaskID     uint // Foreign key from Task
	Name       string
	Path       string
}

func ToResultPathModel(t *models.Task) []ResultPath {
	var results []ResultPath
	for _, dr := range t.ResultPaths {
		r := ResultPath{
			Name: dr.Name,
			Path: dr.Path,
		}
		results = append(results, r)
	}
	return results
}

type ResourceConfig struct {
	CPU    string
	Memory string
	Disk   string
	GPU    string
}

type NetworkConfig struct {
	Type    string
	Domains datatypes.JSON // Using JSON data type for slices
}

type TimeoutConfig struct {
	ExecutionTimeout int64
}

type Execution struct {
	gorm.Model
	ExecutionID        string
	EvaluationID       string
	NodeID             string
	JobID              string
	Namespace          string
	Name               string
	AllocatedResources datatypes.JSON
	DesiredState       ExecutionState   `gorm:"foreignKey:ExecutionID;references:ExecutionID"`
	ComputeState       ExecutionState   `gorm:"foreignKey:ExecutionID;references:ExecutionID"`
	PublishedResult    SpecConfig       `gorm:"embedded;embeddedPrefix:published_"`
	RunOutput          RunCommandResult `gorm:"embedded;embeddedPrefix:run_"`
	PreviousExecution  string
	NextExecution      string
	FollowupEvalID     string
	Revision           uint64
	CreateTime         int64
	ModifiedTime       int64
}

func (e *Execution) AsExecution() models.Execution {
	out := models.Execution{
		ID:        e.ExecutionID,
		Namespace: e.Namespace,
		EvalID:    e.EvaluationID,
		Name:      e.Name,
		NodeID:    e.NodeID,
		JobID:     e.JobID,
		Job:       nil,
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateType(e.DesiredState.State),
			Message:   e.DesiredState.Message,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateType(e.ComputeState.State),
			Message:   e.ComputeState.Message,
		},
		RunOutput: &models.RunCommandResult{
			STDOUT:          e.RunOutput.STDOUT,
			StdoutTruncated: e.RunOutput.StdoutTruncated,
			STDERR:          e.RunOutput.STDERR,
			StderrTruncated: e.RunOutput.StderrTruncated,
			ExitCode:        e.RunOutput.ExitCode,
			ErrorMsg:        e.RunOutput.ErrorMsg,
		},
		PreviousExecution: e.PreviousExecution,
		NextExecution:     e.NextExecution,
		FollowupEvalID:    e.FollowupEvalID,
		Revision:          e.Revision,
		CreateTime:        e.CreateTime,
		ModifyTime:        e.ModifiedTime,
	}
	if e.AllocatedResources != nil {
		var resources models.AllocatedResources
		if err := json.Unmarshal(e.AllocatedResources, &resources); err != nil {
			panic(err)
		}
		out.AllocatedResources = &resources
	}
	if e.PublishedResult.Params != nil {
		var res map[string]interface{}
		if err := json.Unmarshal(e.PublishedResult.Params, &res); err != nil {
			panic(err)
		}
		out.PublishedResult.Params = res
	}
	// out.PublishedResult.Type = e.PublishedResult.Type
	return out
}

type ExecutionState struct {
	gorm.Model
	ExecutionID string
	State       int
	Message     string
}

type RunCommandResult struct {
	STDOUT          string
	StdoutTruncated bool
	STDERR          string
	StderrTruncated bool
	ExitCode        int
	ErrorMsg        string
}

type Evaluation struct {
	gorm.Model
	EvaluationID string
	Namespace    string
	JobID        string
	TriggeredBy  string
	Priority     int
	Type         string
	Status       string
	Comment      string
	WaitUntil    time.Time
	CreatedTime  int64
	ModifiedTime int64
}

func (e Evaluation) AsEvaluation() models.Evaluation {
	return models.Evaluation{
		ID:          e.EvaluationID,
		Namespace:   e.Namespace,
		JobID:       e.JobID,
		TriggeredBy: e.TriggeredBy,
		Priority:    e.Priority,
		Type:        e.Type,
		Status:      e.Status,
		Comment:     e.Comment,
		WaitUntil:   e.WaitUntil,
		CreateTime:  e.CreatedTime,
		ModifyTime:  e.ModifiedTime,
	}
}
