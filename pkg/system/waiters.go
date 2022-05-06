package system

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/docker"
)

// wait for a file to appear that is owned by us
func WaitForFile(path string, maxAttempts int, delay time.Duration) error {
	waiter := &FunctionWaiter{
		Name:        fmt.Sprintf("wait for file to appear: %s", path),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Logging:     true,
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
		Logging:     true,
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

func WaitForContainer(client *docker.DockerClient, id string, maxAttempts int, delay time.Duration) error {
	waiter := &FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Logging:     true,
		Handler: func() (bool, error) {
			container, err := client.GetContainer(id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			return container.State == "running", nil
		},
	}
	return waiter.Wait()
}

func WaitForContainerLogs(client *docker.DockerClient, id string, maxAttempts int, delay time.Duration, findString string) error {
	waiter := &FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Logging:     true,
		Handler: func() (bool, error) {
			container, err := client.GetContainer(id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			if container.State != "running" {
				return false, nil
			}
			logs, err := client.GetLogs(id)
			if err != nil {
				return false, err
			}
			return strings.Contains(logs, findString), nil
		},
	}
	return waiter.Wait()
}
