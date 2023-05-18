package types

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	bacalhau_model "github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	ID             int       `json:"id"`
	Created        time.Time `json:"created"`
	Username       string    `json:"username"`
	HashedPassword string
}

type Counter struct {
	Count int `json:"count"`
}

type AnnotationSummary struct {
	Annotation string `json:"annotation"`
	Count      int    `json:"count"`
}

type JobMonthSummary struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

type JobExecutorSummary struct {
	Executor string `json:"executor"`
	Count    int    `json:"count"`
}

type ModerationType string

const (
	ModerationTypeDatacap   ModerationType = "datacap"
	ModerationTypeExecution ModerationType = "execution"
	ModerationTypeResult    ModerationType = "result"
)

type Moderation struct {
	ID            int64     `json:"id"`
	RequestID     int64     `json:"request_id"`
	UserAccountID int       `json:"user_account_id"`
	Created       time.Time `json:"created"`
	Status        bool      `json:"status"`
	Notes         string    `json:"notes"`
}

type ModerationRequest interface {
	GetID() int64
	GetType() ModerationType
	GetCallback() *URL
}

type ModerationSummary struct {
	Moderation *Moderation       `json:"moderation"`
	Request    ModerationRequest `json:"request"`
	User       *User             `json:"user"`
}

type JobModerationRequest struct {
	ID       int64          `json:"id"`
	Created  time.Time      `json:"created"`
	Callback URL            `json:"callback"`
	JobID    string         `json:"job_id"`
	Type     ModerationType `json:"type"`
}

func (req *JobModerationRequest) GetID() int64 {
	return req.ID
}

func (req *JobModerationRequest) GetType() ModerationType {
	return req.Type
}

func (req *JobModerationRequest) GetCallback() *URL {
	return &req.Callback
}

type JobModerationSummary = ModerationSummary

// A ResultModerationRequest represents a request to moderate a result produced
// by a job.
//
// We explicitly tie the moderation of results to jobs because jobs might be for
// different users with different trust levels, and also because one job
// producing a result that is the same as another job might be revealing
// depending on the execution (i.e. automatically approving a result
// representing the "empty set" might cause problems). So we tie each moderation
// request to a moderation request for a job.
type ResultModerationRequest struct {
	JobModerationRequest
	ExecutionID model.ExecutionID          `json:"execution_id"`
	StorageSpec bacalhau_model.StorageSpec `json:"storage_spec"`
}

type JobInfo struct {
	Job         bacalhau_model.Job               `json:"job"`
	State       bacalhau_model.JobState          `json:"state"`
	Events      []bacalhau_model.JobEvent        `json:"events"`
	Results     []bacalhau_model.PublishedResult `json:"results"`
	Requests    []ModerationRequest              `json:"requests"`
	Moderations []JobModerationSummary           `json:"moderations"`
}

type JobRelation struct {
	JobID string `json:"job_id,omitempty"`
	CID   string `json:"cid,omitempty"`
}

type JobDataIO struct {
	JobID       string `json:"job_id,omitempty"`
	InputOutput string `json:"input_output,omitempty"`
	IsInput     bool   `json:"is_input"`
}
