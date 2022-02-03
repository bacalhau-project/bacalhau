package system

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

func RunCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// same as run command but also returns buffers for stdout and stdin
func RunTeeCommand(command string, args []string) (error, *bytes.Buffer, *bytes.Buffer) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	cmd := exec.Command(command, args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuf)
	return cmd.Run(), stdoutBuf, stderrBuf
}

func RunCommandGetResults(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	result, err := cmd.CombinedOutput()
	return string(result), err
}
