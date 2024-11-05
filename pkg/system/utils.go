package system

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/rs/zerolog"
)

// Making these variable to allow for testing

// MaxStdoutFileLength sets the max size for stdout file during container execution (needed to prevent DoS)
var MaxStdoutFileLength = 1 * datasize.GB

// MaxStderrFileLength sets the max size for stderr file during container execution (needed to prevent DoS)
var MaxStderrFileLength = 1 * datasize.GB

// MaxStdoutReturnLength sets the max size for stdout string return into RunOutput (with truncation)
// from container execution (needed to prevent DoS)
var MaxStdoutReturnLength = 2 * datasize.KB

// MaxStderrReturnLength sets the max size for stderr string return into RunOutput (with truncation)
// from container execution (needed to prevent DoS)
var MaxStderrReturnLength = 2 * datasize.KB

// TODO: #282 we need these to avoid stream based deadlocks
// https://go-review.googlesource.com/c/go/+/42271/3/misc/android/go_android_exec.go#37

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
	r := regexp.MustCompile(`Job ID: (j-[a-f0-9-]{36})`)

	b := r.FindStringSubmatch(testOutput)
	if len(b) > 1 {
		return b[1]
	}
	return ""
}

func FindJobIDInTestOutputLegacy(testOutput string) string {
	// Build a regex starting with Job ID and ending with a UUID
	r := regexp.MustCompile(`Job ID: ([a-f0-9-]{36})`)

	b := r.FindStringSubmatch(testOutput)
	if len(b) > 1 {
		return b[1]
	}
	return ""
}

func MustParseURL(uri string) *url.URL {
	url, err := url.Parse(uri)
	if err != nil {
		panic(fmt.Sprintf("url does not parse: %s", uri))
	}
	return url
}

// IsDebugMode returns true if the environment variable DEBUG is set to true
func IsDebugMode() bool {
	// TODO: #4535 we need to add a flag to the CLI to enable debug mode
	return os.Getenv("DEBUG") == "true" || zerolog.GlobalLevel() <= zerolog.DebugLevel
}

// ExtractCSVData extracts CSV data from the output
func ExtractCSVData(output string) (csvData string, remainingOutput string, err error) {
	lines := strings.Split(output, "\n")

	if len(lines) == 0 {
		return "", "", fmt.Errorf("output is empty")
	}

	// Assume the first line is the header
	headerIndex := 0

	// Find the index where the CSV data ends
	endIndex := len(lines)
	for i, line := range lines[headerIndex+1:] {
		if strings.TrimSpace(line) == "" {
			endIndex = headerIndex + 1 + i
			break
		}
	}

	// Extract the CSV lines
	csvLines := lines[headerIndex:endIndex]
	csvData = strings.Join(csvLines, "\n")

	// Extract the remaining output
	remainingLines := lines[endIndex:]
	remainingOutput = strings.Join(remainingLines, "\n")

	return csvData, remainingOutput, nil
}
