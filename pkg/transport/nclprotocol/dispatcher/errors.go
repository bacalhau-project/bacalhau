package dispatcher

import "fmt"

type ErrDispatcher struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *ErrDispatcher) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("dispatcher error during %s", e.Op)
	}
	return fmt.Sprintf("dispatcher error during %s: %v", e.Op, e.Err)
}

// Error constructors
func newPublishError(err error) error {
	return &ErrDispatcher{Op: "publish", Err: err}
}
