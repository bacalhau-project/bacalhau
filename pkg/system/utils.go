package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var MaxStdoutFileLengthInGB = 1
var MaxStderrFileLengthInGB = 1
var MaxStdoutReturnLengthInBytes = 2048
var MaxStderrReturnLengthInBytes = 2048
var ReadChunkSizeInBytes = 4096

const BufferedWriterSize = 4096

// TODO: #282 we need these to avoid stream based deadlocks
// https://go-review.googlesource.com/c/go/+/42271/3/misc/android/go_android_exec.go#37

var Stdout = struct{ io.Writer }{os.Stdout}
var Stderr = struct{ io.Writer }{os.Stderr}

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

func UnsafeForUserCodeRunCommand(command string, args []string) *model.RunCommandResult {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()
	if err != nil {
		return &model.RunCommandResult{Error: err}
	}
	result := model.NewRunCommandResult()
	result.STDOUT = stdoutBuf.String()
	result.STDERR = stderrBuf.String()
	result.ExitCode = cmd.ProcessState.ExitCode()
	return result
}

func RunCommandResultsToDisk(command string, args []string, stdoutFilename, stderrFilename string) *model.RunCommandResult {
	return runCommandResultsToDisk(command,
		args,
		stdoutFilename,
		stderrFilename,
		MaxStdoutFileLengthInGB,
		MaxStderrFileLengthInGB,
		MaxStdoutReturnLengthInBytes,
		MaxStderrReturnLengthInBytes)
}

// Adding an internal only function to make it easier to test
//
//nolint:funlen // Not sure how to make this shorter without obfuscating functionility
func runCommandResultsToDisk(command string, args []string,
	stdoutFilename string,
	stderrFilename string,
	maxStdoutFileLengthInGB int,
	maxStderrFileLengthInGB int,
	maxStdoutReturnLengthInBytes int,
	maxStderrReturnLengthInBytes int) *model.RunCommandResult {
	// create the return variables ahead of time so we can use them in the goroutine
	r := model.NewRunCommandResult()

	// Setting up variables and command
	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	// Creating output files, file writers, and scanners
	stdoutFileScanner, stdoutFileWriter, stdoutFile, err := createReaderAndWriter(stdoutPipe, stdoutFilename)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating stdout file, writer and scanner: %s", stdoutFilename)
		r.Error = err
		return r
	}

	// Stack in reverse order (sync first, then close - but defers are done LIFO)
	defer func() {
		err = stdoutFile.Close()
		if err != nil {
			log.Error().Err(err).Msgf("Error closing stdout file: %s", stdoutFilename)
		}
	}()

	defer func() {
		err = stdoutFile.Sync()
		if err != nil {
			log.Error().Err(err).Msgf("Error syncing stdout file: %s", stdoutFilename)
		}
	}()

	stderrFileReader, stderrFileWriter, stderrFile, err := createReaderAndWriter(stderrPipe, stderrFilename)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating stderr file, writer and scanner: %s", stderrFilename)
		r.Error = err
		return r
	}

	// Stack in reverse order (sync first, then close - but defers are done LIFO)
	defer func() {
		err = stderrFile.Close()
		if err != nil {
			log.Error().Err(err).Msgf("Error closing stderr file: %s", stderrFilename)
		}
	}()

	defer func() {
		err = stderrFile.Sync()
		if err != nil {
			log.Error().Err(err).Msgf("Error syncing stderr file: %s", stderrFilename)
		}
	}()

	// Go routines for non-blocking reading of stdout and stderr and writing to files
	g := new(errgroup.Group)

	// Read stdout in goroutine.
	g.Go(func() error {
		// TODO: #626 Do we care how exact we are to getting to "Max length"?
		// E.g. if the token pushes us to MaxLength+1 byte, are we ok? Not sure based on how scanning works
		err = writeFromProcessToFileWithMax(stdoutFileScanner, stdoutFileWriter, maxStdoutFileLengthInGB)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing to stdout file: %s", stdoutFilename)
			return err
		}
		return nil
	})

	// Read stderr in goroutine.
	g.Go(func() error {
		// TODO: #626 Do we care how exact we are to getting to "Max length"?
		// E.g. if the token pushes us to MaxLength+1 byte, are we ok? Not sure based on how scanning works
		err = writeFromProcessToFileWithMax(stderrFileReader, stderrFileWriter, maxStderrFileLengthInGB)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing to stderr file: %s", stderrFilename)
			return err
		}
		return nil
	})

	// Starting the command
	if r.Error = cmd.Start(); r.Error != nil {
		log.Error().Err(r.Error).Msg("Error starting command")
		return r
	}

	// Wait the command in a goroutine.
	g.Go(func() error {
		return cmd.Wait()
	})

	// Waiting until errorGroups groups are done
	if r.Error = g.Wait(); r.Error != nil {
		log.Error().Err(r.Error).Msg("Error during running of the command")
	}

	// Reading in stdout and stderr from files
	r.STDOUT, r.Error = readProcessOutputFromFile(stdoutFile, maxStdoutReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error reading stdout from file")
		return r
	}

	r.STDERR, r.Error = readProcessOutputFromFile(stderrFile, maxStderrReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error reading stderr from file")
		return r
	}

	// Reporting if the output for command was truncated in the description
	r.StdoutTruncated, r.Error = wasProcessOutputTruncated(stdoutFilename, maxStdoutReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error checking if stdout was truncated")
		return r
	}

	r.ExitCode = cmd.ProcessState.ExitCode()
	r.Error = nil

	return r
}

func wasProcessOutputTruncated(stdoutFilename string, maxStdoutReturnLengthInBytes int) (bool, error) {
	stdoutFileInfo, err := os.Stat(stdoutFilename)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting file info: %s", stdoutFilename)
		return false, err
	}
	return stdoutFileInfo.Size() > int64(maxStdoutReturnLengthInBytes), nil
}

func readProcessOutputFromFile(f *os.File, maxStdoutReturnLengthInBytes int) (string, error) {
	fb := make([]byte, maxStdoutReturnLengthInBytes)
	_, err := f.Read(fb)
	if err != nil && err != io.EOF {
		log.Error().Err(err).Msgf("Error reading file (though we wrote to it already - weird): %s", f.Name())
		return "", err
	}
	return string(fb), nil
}

func writeFromProcessToFileWithMax(r io.Reader,
	fw *bufio.Writer,
	maxFileLengthInGB int) error {
	currentWrittenLength := int64(0)

	chunk := make([]byte, ReadChunkSizeInBytes) // Size is the chunk size.

	for {
		n, err := io.ReadFull(r, chunk)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Err(err).Msg("Error reading from process:")
			return err
		}

		log.Trace().Msgf("Num of bytes read from process: %d", n)

		if n > 0 {
			var nn int // number of written bytes
			nn, err = fw.Write(chunk[:n])
			if err != nil {
				log.Err(err).Msg("Error writing to output file:")
				return err
			}

			currentWrittenLength += int64(nn)
			if err != nil {
				return err
			}
			fw.Flush()
			if currentWrittenLength > int64(maxFileLengthInGB*int(datasize.GB)) {
				log.Warn().Msgf("Process output file has exceeded the max length of %d GB, stopping...", maxFileLengthInGB)
				fmt.Fprintf(fw, "FILE EXCEEDED MAXIMUM SIZE (%d GB). STOPPING.", maxFileLengthInGB)
				break
			}
		} else {
			break // Num of Bytes is 0, so we are done.
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			log.Trace().Msg("Reached EOF process pipe")
			break
		}
	}
	return nil
}

func createReaderAndWriter(filePipe io.ReadCloser, filename string) (io.Reader, *bufio.Writer, *os.File, error) {
	fileReader := bufio.NewReader(filePipe)
	outputFile, err := os.Create(filename)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating file: %s", filename)
		return nil, nil, nil, err
	}
	stdoutFileWriter := bufio.NewWriter(outputFile)
	return fileReader, stdoutFileWriter, outputFile, nil
}

// TODO: #634 Pretty high priority to allow this to be configurable to a different directory than $HOME/.bacalhau
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

	r := UnsafeForUserCodeRunCommand("mkdir", []string{
		"-p",
		path,
	})
	return path, r.Error
}

func GetResultsDirectory(jobID, hostID string) string {
	return fmt.Sprintf("results/%s/%s", ShortID(jobID), hostID)
}

func ShortID(id string) string {
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

func GetJobStateStringArray(states []model.JobStateType) []string {
	ret := []string{}
	for _, state := range states {
		ret = append(ret, state.String())
	}
	return ret
}

func ShortString(s string, n int) string {
	if len(s) < n {
		return s
	}
	return s[0:n] + "..."
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GetRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))] //nolint:gosec // weak random number is ok
	}
	return string(b)
}

// PathExists returns whether the given file or directory exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
