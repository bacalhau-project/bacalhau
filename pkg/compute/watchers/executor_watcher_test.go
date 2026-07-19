//go:build unit || !integration

package watchers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func TestExecutionUpsertHandlerPropagatesExecutorErrors(t *testing.T) {
	tests := []struct {
		name  string
		state models.ExecutionStateType
		setup func(*compute.MockExecutor, *models.Execution, error)
	}{
		{
			name:  "run",
			state: models.ExecutionStateBidAccepted,
			setup: func(executor *compute.MockExecutor, execution *models.Execution, expectedErr error) {
				executor.EXPECT().Run(gomock.Any(), execution).Return(expectedErr)
			},
		},
		{
			name:  "cancel",
			state: models.ExecutionStateCancelled,
			setup: func(executor *compute.MockExecutor, execution *models.Execution, expectedErr error) {
				executor.EXPECT().Cancel(gomock.Any(), execution).Return(expectedErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			executor := compute.NewMockExecutor(ctrl)
			execution := mock.Execution()
			execution.ComputeState = models.NewExecutionState(tt.state)
			expectedErr := errors.New("executor failure")
			tt.setup(executor, execution, expectedErr)

			handler := NewExecutionUpsertHandler(executor, compute.Bidder{})
			err := handler.HandleEvent(context.Background(), watcher.Event{
				Object: models.ExecutionUpsert{Current: execution},
			})

			require.ErrorIs(t, err, expectedErr)
			require.ErrorContains(t, err, "failed to handle execution state "+tt.state.String())
		})
	}
}
