package model

// Struct to return information about the run.
type RunOutput struct {
	// stdout of the run.
	STDOUT string `json:"stdout"`

	// stderr of the run.
	STDERR string `json:"stderr"`

	// exit code of the run.
	ExitCode int `json:"exitCode"`

	// Runner error
	RunnerError error `json:"runnerError"`
}
