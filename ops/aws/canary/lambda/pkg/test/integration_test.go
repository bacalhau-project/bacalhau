//go:build integration

package test

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/stretchr/testify/require"
)

func TestScenariosAgainstProduction(t *testing.T) {
	for name := range router.TestcasesMap {
		t.Run(name, func(t *testing.T) {
			if name == "submitDockerIPFSJobAndGet" {
				t.Skip("skipping submitDockerIPFSJobAndGet as it is not stable yet. " +
					"https://github.com/filecoin-project/bacalhau/issues/1869")
				return
			}
			event := models.Event{Action: name}
			err := router.Route(context.Background(), event)
			require.NoError(t, err)
		})
	}
}
