package resolver

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type CheckStateFunction func(store.LocalExecutionState) (bool, error)

func CheckForTerminalStates() CheckStateFunction {
	return func(execution store.LocalExecutionState) (bool, error) {
		if execution.State.IsTerminal() {
			return true, nil
		}
		return false, nil
	}
}

func CheckForState(expectedStates ...store.LocalExecutionStateType) CheckStateFunction {
	return func(execution store.LocalExecutionState) (bool, error) {
		for _, expectedState := range expectedStates {
			if execution.State == expectedState {
				return true, nil
			}
		}
		return false, nil
	}
}

func CheckForUnexpectedState(expectedStates ...store.LocalExecutionStateType) CheckStateFunction {
	return func(execution store.LocalExecutionState) (bool, error) {
		for _, expectedState := range expectedStates {
			if execution.State == expectedState {
				return false, fmt.Errorf("unexpected state: %s", execution.State)
			}
		}
		return false, nil
	}
}

func CheckCompleted() CheckStateFunction {
	return CheckForState(store.ExecutionStateCompleted)
}
