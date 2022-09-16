package model

// Struct to return information about the run.
type RunExecutorResult struct {
	// stdout of the run.
	STDOUT string `json:"stdout"`

	// bool describing if stdout was truncated
	StdoutTruncated bool `json:"stdouttruncated"`

	// stderr of the run.
	STDERR string `json:"stderr"`

	// bool describing if stderr was truncated
	StderrTruncated bool `json:"stderrtruncated"`

	// exit code of the run.
	ExitCode int `json:"exitCode"`

	// Runner error
	RunnerError error `json:"runnerError"`
}
