package types

import (
	"time"

	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
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

type JobModeration struct {
	ID            int       `json:"id"`
	JobID         string    `json:"job_id"`
	UserAccountID int       `json:"user_account_id"`
	Created       time.Time `json:"created"`
	Status        string    `json:"status"`
	Notes         string    `json:"notes"`
}

type JobModerationSummary struct {
	Moderation *JobModeration `json:"moderation"`
	User       *User          `json:"user"`
}

type JobInfo struct {
	Job        bacalhau_model.Job               `json:"job"`
	State      bacalhau_model.JobState          `json:"state"`
	Events     []bacalhau_model.JobEvent        `json:"events"`
	Results    []bacalhau_model.PublishedResult `json:"results"`
	Moderation JobModerationSummary             `json:"moderation"`
}
