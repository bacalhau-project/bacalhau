//go:build integration && !windows

package devstack

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const minimumOpenFiles int = 512

//nolint:unused // golangci-lint complains that this is unused, but it's in pkg/test/devstack/sharding_test.go
func shouldRunShardingTest() (bool, error) {
	ulimitValue := 0

	if _, err := exec.LookPath("ulimit"); err == nil {
		// Test to see how many files can be open on this system...
		cmd := exec.Command("ulimit", "-n")
		out, err := cmd.Output()
		if err != nil {
			return false, err
		}

		ulimitValue, err = strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			return false, err
		}
	} else {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return false, err
		}
		ulimitValue, err = strconv.Atoi(fmt.Sprint(rLimit.Cur))
		if err != nil {
			return false, err
		}
	}

	return ulimitValue > minimumOpenFiles, nil
}
