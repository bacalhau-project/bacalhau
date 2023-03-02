package v1beta1

type RunCommandResult struct {
	// stdout of the run. Yaml provided for `describe` output
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
	ErrorMsg string `json:"runnerError"`
}

func NewRunCommandResult() *RunCommandResult {
	return &RunCommandResult{
		STDOUT:          "",    // stdout of the run.
		StdoutTruncated: false, // bool describing if stdout was truncated
		STDERR:          "",    // stderr of the run.
		StderrTruncated: false, // bool describing if stderr was truncated
		ExitCode:        -1,    // exit code of the run.
	}
}
