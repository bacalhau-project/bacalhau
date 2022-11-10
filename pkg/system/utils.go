package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"golang.org/x/exp/constraints"

	"github.com/rs/zerolog/log"
)

// Making these variable to allow for testing

// MaxStdoutFileLengthInBytes sets the max size for stdout file during container execution (needed to prevent DoS)
var MaxStdoutFileLengthInBytes = int(1 * datasize.GB)

// MaxStderrFileLengthInBytes sets the max size for stderr file during container execution (needed to prevent DoS)
var MaxStderrFileLengthInBytes = int(1 * datasize.GB)

// MaxStdoutReturnLengthInBytes sets the max size for stdout string return into RunOutput (with trunctation)
// from container execution (needed to prevent DoS)
var MaxStdoutReturnLengthInBytes = 2048

// MaxStderrReturnLengthInBytes sets the max size for stderr string return into RunOutput (with trunctation)
// from container execution (needed to prevent DoS)
var MaxStderrReturnLengthInBytes = 2048

const ReadChunkSizeInBytes = 1024

// TODO: #282 we need these to avoid stream based deadlocks
// https://go-review.googlesource.com/c/go/+/42271/3/misc/android/go_android_exec.go#37

var Stdout = struct{ io.Writer }{os.Stdout}
var Stderr = struct{ io.Writer }{os.Stderr}

func UnsafeForUserCodeRunCommand(command string, args []string) (*model.RunCommandResult, error) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	log.Trace().Msgf("Command: %s %s", command, args)
	cmd := exec.Command(command, args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()
	if err != nil {
		return &model.RunCommandResult{ErrorMsg: err.Error()}, err
	}
	result := model.NewRunCommandResult()
	result.STDOUT = stdoutBuf.String()
	result.STDERR = stderrBuf.String()
	result.ExitCode = cmd.ProcessState.ExitCode()
	return result, nil
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
		r.ErrorMsg = err.Error()
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
		r.ErrorMsg = err.Error()
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
	if err = cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Error starting command")
		r.ErrorMsg = err.Error()
		return r, err
	}

	// Wait the command in a goroutine.
	wg.Wait()
	if err = cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("Error during running of the command")
		r.ErrorMsg = err.Error()
		return r, err
	}

	// Reading in stdout and stderr from files
	r.STDOUT, r.StdoutTruncated, err = readProcessOutputFromFile(stdoutFile, maxStdoutReturnLengthInBytes)
	if err != nil {
		log.Error().Err(err).Msg("Error reading stdout from file")
		r.ErrorMsg = err.Error()
		return r, err
	}

	r.STDERR, r.StderrTruncated, err = readProcessOutputFromFile(stderrFile, maxStderrReturnLengthInBytes)
	if err != nil {
		log.Error().Err(err).Msg("Error reading stderr from file")
		r.ErrorMsg = err.Error()
		return r, err
	}
	if strings.Contains(r.STDERR, "executable file not found") {
		err = bacerrors.NewExecutableNotFound(strings.Join(args[1:], " "))
		r.STDERR = err.Error()
	}

	r.ExitCode = cmd.ProcessState.ExitCode()
	return r, err
}

func readProcessOutputFromFile(f *os.File, maxVariableReturnLengthInBytes int) (output string, isTruncated bool, err error) {
	log.Trace().Msgf("Reading from file: %s", f.Name())

	// start by resetting the file seek position
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		log.Error().Err(err).Msgf("Error seeking to beginning of file: %s", f.Name())
		return "", isTruncated, err
	}

	fileStat, err := f.Stat()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting file info: %s", f.Name())
		return "", isTruncated, err
	}
	fileSize := fileStat.Size()
	amountToRead := int64(math.Min(float64(maxVariableReturnLengthInBytes), float64(fileSize)))

	fb := make([]byte, amountToRead)

	// If the file is larger than the max, we only read the last max bytes
	if fileSize > amountToRead {
		isTruncated = true
		_, err = f.Seek(fileSize-amountToRead, 0)
		if err != nil {
			log.Error().Err(err).Msgf("Error seeking to end of file: %s", f.Name())
			return "", isTruncated, err
		}
	}

	_, err = f.Read(fb)
	if err != nil && err != io.EOF {
		log.Error().Err(err).Msgf("Error reading file (though we wrote to it already - weird): %s", f.Name())
		return "", isTruncated, err
	}
	return strings.TrimSpace(string(fb)), isTruncated, nil
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

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func ReverseList(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func SplitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func FindJobIDInTestOutput(testOutput string) string {
	// Build a regex starting with Job ID and ending with a UUID
	r := regexp.MustCompile(`Job ID: ([a-f0-9-]{36})`)

	b := r.FindStringSubmatch(testOutput)
	if len(b) > 1 {
		return b[1]
	}
	return ""
}

func GetShortID(ID string) string {
	if len(ID) < model.ShortIDLength {
		return ID
	}
	return ID[:model.ShortIDLength]
}
