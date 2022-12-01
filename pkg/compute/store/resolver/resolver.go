package resolver

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

var DefaultStateResolverParams = StateResolverParams{
	MaxWaitAttempts: 500, // 10 seconds total wait time
	WaitDelay:       20 * time.Millisecond,
}

type StateResolverParams struct {
	ExecutionStore  store.ExecutionStore
	MaxWaitAttempts int
	WaitDelay       time.Duration
}
type StateResolver struct {
	executionStore  store.ExecutionStore
	maxWaitAttempts int
	waitDelay       time.Duration
}

func NewStateResolver(params StateResolverParams) *StateResolver {
	if params.MaxWaitAttempts == 0 {
		params.MaxWaitAttempts = DefaultStateResolverParams.MaxWaitAttempts
	}
	if params.WaitDelay == 0 {
		params.WaitDelay = DefaultStateResolverParams.WaitDelay
	}
	return &StateResolver{
		executionStore:  params.ExecutionStore,
		maxWaitAttempts: params.MaxWaitAttempts,
		waitDelay:       params.WaitDelay,
	}
}

func (r *StateResolver) Wait(
	ctx context.Context,
	executionID string,
	checkStateFunctions ...CheckStateFunction) error {
	waiter := &system.FunctionWaiter{
		Name:        "Wait for execution state",
		MaxAttempts: r.maxWaitAttempts,
		Delay:       r.waitDelay,
		Handler: func() (bool, error) {
			execution, err := r.executionStore.GetExecution(ctx, executionID)
			if err != nil {
				return false, err
			}

			allOK := true
			for _, checkFunction := range checkStateFunctions {
				stepOK, stepErr := checkFunction(execution)
				if stepErr != nil {
					return false, stepErr
				}
				allOK = allOK && stepOK
			}

			if allOK {
				return true, nil
			}

			allTerminal, err := CheckForTerminalStates()(execution)
			if err != nil {
				return false, err
			}
			if allTerminal {
				return false, fmt.Errorf(
					"execution reached a terminal state before meeting the resolver's conditions: %s", execution)
			}
			return false, nil
		},
	}
	return waiter.Wait(ctx)
}
