package testutils

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetJobFromTestOutput(ctx context.Context, t *testing.T, c *client.APIClient, out string) model.Job {
	jobID := system.FindJobIDInTestOutput(out)
	uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	require.Regexp(t, uuidRegex, jobID, "Job ID should be a UUID")

	j, _, err := c.Get(ctx, jobID)
	require.NoError(t, err)
	require.NotNil(t, j, "Failed to get job with ID: %s", out)
	return j.Job
}

func FirstFatalError(_ *testing.T, output string) (model.TestFatalErrorHandlerContents, error) {
	linesInOutput := system.SplitLines(output)
	fakeFatalError := &model.TestFatalErrorHandlerContents{}
	for _, line := range linesInOutput {
		err := marshaller.JSONUnmarshalWithMax([]byte(line), fakeFatalError)
		if err != nil {
			return model.TestFatalErrorHandlerContents{}, err
		} else {
			return *fakeFatalError, nil
		}
	}
	return model.TestFatalErrorHandlerContents{}, fmt.Errorf("no fatal error found in output")
}

func MakeNoopJob(t testing.TB) *model.Job {
	j := MakeJobWithOpts(t)
	return &j
}

func MakeJobWithOpts(t testing.TB, opts ...job.SpecOpt) model.Job {
	spec, err := job.MakeSpec(opts...)
	if err != nil {
		t.Fatalf("creating job spec: %s", err)
	}

	j := model.NewJob()
	j.Spec = spec
	return *j
}

func MakeSpecWithOpts(t testing.TB, opts ...job.SpecOpt) model.Spec {
	spec, err := job.MakeSpec(opts...)
	if err != nil {
		t.Fatalf("creating job spec: %s", err)
	}
	return spec
}
