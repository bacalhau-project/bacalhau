package system

import (
	"fmt"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func CheckBashSyntax(cmds []string) error {
	script := strings.NewReader(strings.Join(cmds, "\n"))
	_, err := syntax.NewParser().Parse(script, "")

	return err
}

// Function for validating the workdir of a docker command.
func ValidateWorkingDir(jobWorkingDir string) error {
	if jobWorkingDir != "" {
		if !filepath.IsAbs(jobWorkingDir) {
			return fmt.Errorf("workdir must be an absolute path. Passed in: %s", jobWorkingDir)
		}
	}
	return nil
}
