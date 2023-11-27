package supervisor

import "fmt"

func ErrNoEngine(engine string) error {
	return fmt.Errorf("plugin not found '%s'", engine)
}

func ErrExecutionNotSupervised(executionID string) error {
	return fmt.Errorf("execution not supervised '%s'", executionID)
}
