package system

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func CommandLogger(command string, args []string) {
	if os.Getenv("DEBUG") == "" {
		return
	}
	fmt.Printf("----------------------------------\nRunning command: %s %s\n----------------------------------\n", command, strings.Join(args, " "))
}

func RunCommand(command string, args []string) error {
	CommandLogger(command, args)
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// same as run command but also returns buffers for stdout and stdin
func RunTeeCommand(command string, args []string) (error, *bytes.Buffer, *bytes.Buffer) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	CommandLogger(command, args)
	cmd := exec.Command(command, args...)
	cmd.Stdout = io.MultiWriter(os.Stdout, stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuf)
	return cmd.Run(), stdoutBuf, stderrBuf
}

func TryUntilSucceedsN(f func() error, desc string, retries int) error {
	attempt := 0
	for {
		err := f()
		if err != nil {
			if attempt > retries {
				return err
			} else {
				fmt.Printf("Error %s: %v, pausing and trying again...\n", desc, err)
				time.Sleep(time.Duration(attempt) * time.Second)
			}
		} else {
			return nil
		}
		attempt++
	}
}

func RunCommandGetResults(command string, args []string) (string, error) {
	CommandLogger(command, args)
	cmd := exec.Command(command, args...)
	result, err := cmd.CombinedOutput()
	return string(result), err
}

func RunCommandGetResultsEnv(command string, args []string, env []string) (string, error) {
	CommandLogger(command, args)
	cmd := exec.Command(command, args...)
	cmd.Env = env
	result, err := cmd.CombinedOutput()
	return string(result), err
}

func GetSystemDirectory(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/.bacalhau/%s", homeDir, path), nil
}

func EnsureSystemDirectory(path string) (string, error) {
	path, err := GetSystemDirectory(path)
	if err != nil {
		return "", err
	}
	err = RunCommand("mkdir", []string{
		"-p",
		path,
	})
	return path, err
}

func GetResultsDirectory(jobId, hostId string) string {
	return fmt.Sprintf("results/%s/%s", ShortId(jobId), hostId)
}

func ShortId(id string) string {
	parts := strings.Split(id, "-")
	return parts[0]
}

func MapStringArray(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func MapByteArray(vs []byte, f func(byte) byte) []byte {
	vsm := make([]byte, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}