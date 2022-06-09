package system

import (
	"fmt"
	"os"
	"time"
)

// wait for a file to appear that is owned by us
func WaitForFile(path string, maxAttempts int, delay time.Duration) error {
	waiter := &FunctionWaiter{
		Name:        fmt.Sprintf("wait for file to appear: %s", path),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			_, err := os.Stat(path)
			if err != nil {
				return false, err
			}
			return true, nil
		},
	}

	return waiter.Wait()
}

// wait for a file to appear that needs sudo to read it
// (for example fuse mounted files mounted by a docker container)
func WaitForFileSudo(path string, maxAttempts int, delay time.Duration) error {
	waiter := &FunctionWaiter{
		Name:        fmt.Sprintf("wait for file to appear: %s", path),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			result, err := RunCommandGetResults("sudo", []string{
				"ls", "-la",
				path,
			})

			fmt.Printf("AFTER CHECK: %s %+v\n", result, err)
			if err != nil {
				return false, err
			}
			return true, nil
		},
	}

	return waiter.Wait()
}
