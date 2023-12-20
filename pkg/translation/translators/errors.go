package translators

import "fmt"

func ErrMissingParameters(trs string) error {
	return fmt.Errorf("missing parameters in task for '%s' translator", trs)
}
