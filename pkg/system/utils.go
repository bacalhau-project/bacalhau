package system

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

// we need these to avoid stream based deadlocks
// https://go-review.googlesource.com/c/go/+/42271/3/misc/android/go_android_exec.go#37
var Stdout = struct{ io.Writer }{os.Stdout}
var Stderr = struct{ io.Writer }{os.Stderr}

func RunCommand(command string, args []string) error {
	log.Trace().Msgf(`Command: %s %s`, command, args)
	cmd := exec.Command(command, args...)
	cmd.Stderr = Stderr
	cmd.Stdout = Stdout
	return cmd.Run()
}

// same as run command but also returns buffers for stdout and stdin
func RunTeeCommand(command string, args []string) (error, *bytes.Buffer, *bytes.Buffer) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)

	cmd.Stdout = io.MultiWriter(Stdout, stdoutBuf)
	cmd.Stderr = io.MultiWriter(Stderr, stderrBuf)
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
				log.Trace().Msgf("Error %s: %v, pausing and trying again...", desc, err)
				time.Sleep(1 * time.Second)
			}
		} else {
			return nil
		}
		attempt++
	}
}

func RunCommandGetResults(command string, args []string) (string, error) {
	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	result, err := cmd.CombinedOutput()
	return string(result), err
}

func RunCommandGetResultsEnv(command string, args []string, env []string) (string, error) {
	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	cmd.Env = env
	result, err := cmd.CombinedOutput()
	return string(result), err
}

// TODO: Pretty high priority to allow this to be configurable to a different directory than $HOME/.bacalhau
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

	log.Trace().Msgf("Enforcing creation of results dir: %s", path)

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

func StringArrayContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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

func IsDebug() bool {
	return os.Getenv("LOG_LEVEL") == "debug"
}

func ShouldKeepStack() bool {
	return os.Getenv("KEEP_STACK") != ""
}

func GetJobStateStringArray(states []types.JobStateType) []string {
	ret := []string{}
	for _, state := range states {
		ret = append(ret, string(state))
	}
	return ret
}
