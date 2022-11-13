package system

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"golang.org/x/exp/constraints"
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
