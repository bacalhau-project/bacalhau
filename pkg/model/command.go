package model

type RunCommandResult struct {
	// stdout of the run. Yaml provided for `describe` output
	STDOUT string `json:"stdout" yaml:"Stdout"`

	// bool describing if stdout was truncated
	StdoutTruncated bool `json:"stdouttruncated" yaml:"StdoutTruncated"`

	// stderr of the run.
	STDERR string `json:"stderr" yaml:"Stderr"`

	// bool describing if stderr was truncated
	StderrTruncated bool `json:"stderrtruncated" yaml:"StderrTruncated"`

	// exit code of the run.
	ExitCode int `json:"exitCode" yaml:"ExitCode"`

	// Runner error
	ErrorMsg string `json:"runnerError" yaml:"RunnerError"`
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
