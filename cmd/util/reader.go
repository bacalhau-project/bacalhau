package util

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
)

func ReadJobFromUser(cmd *cobra.Command, args []string) ([]byte, error) {
	// read the job spec from stdin or file
	var err error
	var jobBytes []byte
	if len(args) == 0 {
		jobBytes, err = ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return nil, fmt.Errorf("reading job from stdin: %w", err)
		}
	} else {
		jobBytes, err = ReadFromFile(args[0])
		if err != nil {
			return nil, err
		}
	}
	if len(jobBytes) == 0 {
		return nil, fmt.Errorf("%s: no content provided", userstrings.JobSpecBad)
	}
	return jobBytes, nil
}

func ReadFromStdinIfAvailable(cmd *cobra.Command) ([]byte, error) {
	// write to stderr since stdout is reserved for details of jobs.
	if _, err := io.WriteString(cmd.ErrOrStderr(), "Reading from /dev/stdin; send Ctrl-d to stop."); err != nil {
		return nil, fmt.Errorf("unable to write to stderr: %w", err)
	}
	result, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, fmt.Errorf("reading job from stdin: %w", err)
	}
	return result, nil
}

func ReadFromFile(path string) ([]byte, error) {
	var fileContent *os.File
	fileContent, err := os.Open(path) //nolint:gosec // G304: path parameter validated by caller
	if err != nil {
		return nil, fmt.Errorf("opening job file (%q): %w", path, err)
	}
	defer func() { _ = fileContent.Close() }()

	result, err := io.ReadAll(fileContent)
	if err != nil {
		return nil, fmt.Errorf("reading job file (%q): %w", path, err)
	}
	return result, nil
}
