package scenario

import (
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A CheckSubmitResponse is a function that will examine and validate submitJob response.
// Useful when validating that a job should be rejected.
type CheckSubmitResponse func(job *model.Job, err error) error

// SubmitJobSuccess returns a CheckSubmitResponse that asserts no error was returned when submitting a job.
func SubmitJobSuccess() CheckSubmitResponse {
	return func(job *model.Job, err error) error {
		if err != nil {
			return fmt.Errorf("expected no error, got %v", err)
		}
		if job == nil {
			return fmt.Errorf("expected job, got nil")
		}
		return nil
	}
}

// SubmitJobFail returns a CheckSubmitResponse that asserts an error was returned when submitting a job.
func SubmitJobFail() CheckSubmitResponse {
	return func(job *model.Job, err error) error {
		if err == nil {
			return fmt.Errorf("expected error, got nil")
		}
		return nil
	}
}

func SubmitJobErrorContains(msg string) CheckSubmitResponse {
	return func(job *model.Job, err error) error {
		e := SubmitJobFail()(job, err)
		if e != nil {
			return e
		}

		if !strings.Contains(err.Error(), msg) {
			return fmt.Errorf("expected error to contain %q, got %q", msg, err.Error())
		}
		return nil
	}
}
