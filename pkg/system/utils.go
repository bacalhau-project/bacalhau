package system

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

var MaxStdoutFileLengthInBytes = int(1 * datasize.GB)
var MaxStderrFileLengthInBytes = int(1 * datasize.GB)
var MaxStdoutReturnLengthInBytes = 2048
var MaxStderrReturnLengthInBytes = 2048
var ReadChunkSizeInBytes = 1024

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

func RunCommandResultsToDisk(command string, args []string, stdoutFilename, stderrFilename string) (
	*model.RunCommandResult, error) {
	return runCommandResultsToDisk(command,
		args,
		stdoutFilename,
		stderrFilename,
		MaxStdoutFileLengthInBytes,
		MaxStderrFileLengthInBytes,
		MaxStdoutReturnLengthInBytes,
		MaxStderrReturnLengthInBytes)
}

// Adding an internal only function to make it easier to test
//
//nolint:funlen // Not sure how to make this shorter without obfuscating functionility
func runCommandResultsToDisk(command string, args []string,
	stdoutFilename string,
	stderrFilename string,
	maxStdoutFileLengthInBytes int,
	maxStderrFileLengthInBytes int,
	maxStdoutReturnLengthInBytes int,
	maxStderrReturnLengthInBytes int) (*model.RunCommandResult, error) {
	// create the return variables ahead of time so we can use them in the goroutine
	r := model.NewRunCommandResult()

	// Setting up variables and command
	log.Debug().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	// Creating output files, file writers, and scanners
	stdoutFileReader, stdoutFileWriter, stdoutFile, err := createReaderAndWriter(stdoutPipe, stdoutFilename)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating stdout file, writer and scanner: %s", stdoutFilename)
		r.Error = err
		return r, err
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
		return r, err
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
	var wg sync.WaitGroup
	wg.Add(2)

	// Read stdout in goroutine.
	var stdoutErr error
	go func() {
		// TODO: #626 Do we care how exact we are to getting to "Max length"?
		// E.g. if the token pushes us to MaxLength+1 byte, are we ok? Not sure based on how scanning works
		stdoutErr = writeFromProcessToFileWithMax("stdout",
			stdoutFileReader,
			stdoutFileWriter,
			maxStdoutFileLengthInBytes)
		if stdoutErr != nil {
			log.Error().Err(stdoutErr).Msgf("Error writing to stdout file: %s", stdoutFilename)
		}
		wg.Done()
	}()

	// Read stderr in goroutine.
	var stderrErr error
	go func() {
		// E.g. if the token pushes us to MaxLength+1 byte, are we ok? Not sure based on how scanning works
		stderrErr = writeFromProcessToFileWithMax("stderr",
			stderrFileReader,
			stderrFileWriter,
			maxStderrFileLengthInBytes)
		if stderrErr != nil {
			log.Error().Err(err).Msgf("Error writing to stderr file: %s", stderrFilename)
		}
		wg.Done()
	}()

	// Starting the command
	if r.Error = cmd.Start(); r.Error != nil {
		log.Error().Err(r.Error).Msg("Error starting command")
		return r, err
	}

	// Wait the command in a goroutine.
	wg.Wait()
	r.Error = cmd.Wait()

	// Waiting until errorGroups groups are done
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error during running of the command")
	}

	// Reading in stdout and stderr from files
	r.STDOUT, r.Error = readProcessOutputFromFile(stdoutFile, maxStdoutReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error reading stdout from file")
		return r, err
	}

	r.STDERR, r.Error = readProcessOutputFromFile(stderrFile, maxStderrReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error reading stderr from file")
		return r, err
	}

	// Reporting if the output for command was truncated in the description
	r.StdoutTruncated, r.Error = wasProcessOutputTruncated(stdoutFilename, maxStdoutReturnLengthInBytes)
	if r.Error != nil {
		log.Error().Err(r.Error).Msg("Error checking if stdout was truncated")
		return r, err
	}

	r.ExitCode = cmd.ProcessState.ExitCode()
	r.Error = nil

	return r, err
}

func wasProcessOutputTruncated(filename string, maxVariableReturnLengthInBytes int) (bool, error) {
	stdoutFileInfo, err := os.Stat(filename)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting file info: %s", filename)
		return false, err
	}
	return stdoutFileInfo.Size() > int64(maxVariableReturnLengthInBytes), nil
}

func readProcessOutputFromFile(f *os.File, maxVariableReturnLengthInBytes int) (string, error) {
	log.Debug().Msgf("Reading from file: %s", f.Name())

	_, err := f.Seek(0, 0) // reset to zero
	if err != nil {
		log.Error().Err(err).Msgf("Error seeking to beginning of file: %s", f.Name())
		return "", err
	}

	fileStat, err := f.Stat()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting file info: %s", f.Name())
		return "", err
	}
	fileSize := fileStat.Size()
	amountToRead := int(math.Min(float64(maxVariableReturnLengthInBytes), float64(fileSize)))

	fb := make([]byte, amountToRead)

	// If the file is larger than the max, we only read the last max bytes
	if fileSize > int64(maxVariableReturnLengthInBytes) {
		_, err = f.Seek(fileSize-int64(maxVariableReturnLengthInBytes), 0)
		if err != nil {
			log.Error().Err(err).Msgf("Error seeking to end of file: %s", f.Name())
			return "", err
		}
	}

	_, err = f.Read(fb)
	if err != nil && err != io.EOF {
		log.Error().Err(err).Msgf("Error reading file (though we wrote to it already - weird): %s", f.Name())
		return "", err
	}
	return string(fb), nil
}

func writeFromProcessToFileWithMax(name string, r *bufio.Reader,
	fw *bufio.Writer,
	maxFileLengthInBytes int) error {
	currentWrittenLength := float32(0)
	buf := make([]byte, ReadChunkSizeInBytes)

	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			log.Err(err).Msgf("%s: Error reading from %s pipe", name, err)
		}

		log.Debug().Msgf("DEBUG %s: Read %d bytes from process", name, n)
		log.Debug().Msgf("DEBUG %s: Error from reading from process: %v", name, err)

		if n > 0 {
			var nn int // written bytes
			var bufSizeToWrite int

			if currentWrittenLength+float32(n) > float32(maxFileLengthInBytes) {
				bufSizeToWrite = maxFileLengthInBytes - int(currentWrittenLength)
			} else {
				bufSizeToWrite = n
			}
			nn, err = fw.Write(buf[:bufSizeToWrite])
			if err != nil {
				return err
			}
			fw.Flush()

			currentWrittenLength += float32(nn)

			if int(currentWrittenLength) > maxFileLengthInBytes {
				maxSizeInGB := float32(maxFileLengthInBytes) / float32(datasize.GB)
				log.Warn().Msgf("Process output file has exceeded the max length of %f GB, stopping...", maxSizeInGB)
				fmt.Fprintf(fw, "FILE EXCEEDED MAXIMUM SIZE (%f GB). STOPPING.", maxSizeInGB)
				break
			}
		}

		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				log.Debug().Msgf("%s: EOF detected", name)
				err = nil
			} else {
				log.Err(err).Msgf("%s: Error reading file, non-EOF", name)
			}

			return err
		}
	}

	return nil
}

func createReaderAndWriter(filePipe io.ReadCloser, filename string) (*bufio.Reader, *bufio.Writer, *os.File, error) {
	fileReader := bufio.NewReader(filePipe)
	outputFile, err := os.Create(filename)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating file: %s", filename)
		return nil, nil, nil, err
	}
	fileWriter := bufio.NewWriter(outputFile)
	return fileReader, fileWriter, outputFile, nil
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

func RepeatedCharactersBashCommandToStdout(num int) string {
	return repeatedCharactersBashCommand(num, false)
}

func RepeatedCharactersBashCommandToStderr(num int) string {
	return repeatedCharactersBashCommand(num, true)
}

// Repeats '=' num times. toStderr true == stderr, false == stdout
func repeatedCharactersBashCommand(num int, toStderr bool) string {
	toStderrString := ""

	// If going to stderr, we need to use the special bash command to write to stderr
	if toStderr {
		toStderrString = "| cat 1>&2"
	}
	return fmt.Sprintf(`for i in $(seq 1 %d) ; do echo -n "="; done %s`, num, toStderrString)
}

// TODO: #233 Replace when we move to go1.18
// https://stackoverflow.com/questions/27516387/what-is-the-correct-way-to-find-the-min-between-two-integers-in-go
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MinFromArray(intArray []int) (int, error) {
	if len(intArray) == 0 {
		return 0, errors.New("cannot get min from empty array")
	}

	m := math.MaxInt
	for i, e := range intArray {
		if i == 0 || e < m {
			m = e
		}
	}
	return m, nil
}

func ReverseList(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
