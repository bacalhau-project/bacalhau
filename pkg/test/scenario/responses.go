package scenario

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// A CheckSubmitResponse is a function that will examine and validate submitJob response.
// Useful when validating that a job should be rejected.
type CheckSubmitResponse func(response *apimodels.PutJobResponse, err error) error

// SubmitJobSuccess returns a CheckSubmitResponse that asserts no error was returned when submitting a job.
func SubmitJobSuccess() CheckSubmitResponse {
	return func(response *apimodels.PutJobResponse, err error) error {
		if err != nil {
			return fmt.Errorf("expected no error, got %v", err)
		}
		if response == nil {
			return fmt.Errorf("expected job response, got nil")
		}
		if len(response.Warnings) > 0 {
			return fmt.Errorf("unexpted warnings returned when submitting job: %v", response.Warnings)
		}
		return nil
	}
}

// SubmitJobFail returns a CheckSubmitResponse that asserts an error was returned when submitting a job.
func SubmitJobFail() CheckSubmitResponse {
	return func(_ *apimodels.PutJobResponse, err error) error {
		if err == nil {
			return fmt.Errorf("expected error, got nil")
		}
		return nil
	}
}

func SubmitJobErrorContains(msg string) CheckSubmitResponse {
	return func(response *apimodels.PutJobResponse, err error) error {
		e := SubmitJobFail()(response, err)
		if e != nil {
			return e
		}

		if !strings.Contains(err.Error(), msg) {
			return fmt.Errorf("expected error to contain %q, got %q", msg, err.Error())
		}
		return nil
	}
}
