// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

// this can be extended with params but it's intent is to be a
// "give me any old docker run spec now" function
func DockerRunJob() model.JobSpec {
	return model.JobSpec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"/bin/bash",
				"-c",
				"echo hello",
			},
		},
	}
}
