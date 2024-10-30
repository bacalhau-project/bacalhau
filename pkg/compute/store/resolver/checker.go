package resolver

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type CheckStateFunction func(*models.Execution) (bool, error)

func CheckForTerminalStates() CheckStateFunction {
	return func(execution *models.Execution) (bool, error) {
		if execution.IsTerminalComputeState() {
			return true, nil
		}
		return false, nil
	}
}

func CheckForState(expectedStates ...models.ExecutionStateType) CheckStateFunction {
	return func(execution *models.Execution) (bool, error) {
		for _, expectedState := range expectedStates {
			if execution.ComputeState.StateType == expectedState {
				return true, nil
			}
		}
		return false, nil
	}
}

func CheckForUnexpectedState(expectedStates ...models.ExecutionStateType) CheckStateFunction {
	return func(execution *models.Execution) (bool, error) {
		for _, expectedState := range expectedStates {
			if execution.ComputeState.StateType == expectedState {
				return false, fmt.Errorf("unexpected state: %s", execution.ComputeState.StateType.String())
			}
		}
		return false, nil
	}
}

func CheckCompleted() CheckStateFunction {
	return CheckForState(models.ExecutionStateCompleted)
}
