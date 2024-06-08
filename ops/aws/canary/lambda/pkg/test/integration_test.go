//go:build integration || !unit

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/router"
)

func TestScenariosAgainstProduction(t *testing.T) {
	for name := range router.TestcasesMap {
		t.Run(name, func(t *testing.T) {
			event := models.Event{Action: name}
			err := router.Route(context.Background(), event)
			require.NoError(t, err)
		})
	}
}
