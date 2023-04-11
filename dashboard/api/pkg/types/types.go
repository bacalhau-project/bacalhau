package types

import (
	"time"

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
)

type JobModerationRequest struct {
	ID       int64          `json:"id"`
	JobID    string         `json:"job_id"`
	Type     ModerationType `json:"type"`
	Created  time.Time      `json:"created"`
	Callback URL            `json:"callback"`
}

type JobModeration struct {
	ID            int64     `json:"id"`
	RequestID     int64     `json:"request_id"`
	UserAccountID int       `json:"user_account_id"`
	Created       time.Time `json:"created"`
	Status        bool      `json:"status"`
	Notes         string    `json:"notes"`
}

type JobModerationSummary struct {
	Moderation *JobModeration        `json:"moderation"`
	Request    *JobModerationRequest `json:"request"`
	User       *User                 `json:"user"`
}

type JobInfo struct {
	Job         bacalhau_model.Job               `json:"job"`
	State       bacalhau_model.JobState          `json:"state"`
	Events      []bacalhau_model.JobEvent        `json:"events"`
	Results     []bacalhau_model.PublishedResult `json:"results"`
	Requests    []JobModerationRequest           `json:"requests"`
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
