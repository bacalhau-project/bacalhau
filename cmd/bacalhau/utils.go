package bacalhau

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var listOutputFormat string
var tableOutputWide bool
var tableHideHeader bool
var tableMaxJobs int
var tableSortBy ColumnEnum
var tableSortReverse bool
var tableIdFilter string
var tableNoStyle bool

// var tableMergeValues bool

func shortenTime(t time.Time) string {
	if tableOutputWide {
		return t.Format("06-01-02-15:04:05")
	}

	return t.Format("15:04:05")

}

func shortenString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < 20 {
		return st
	}

	return st[:20] + "..."
}

func shortId(id string) string {
	return id[:8]
}

func getJobResult(job *executor.Job, state *executor.JobState) string {
	if state.ResultsId == "" {
		return ""
	}
	return "/" + job.Spec.Verifier.String() + "/" + state.ResultsId
}

func getAPIClient() *publicapi.APIClient {
	return publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

func ExecuteTestCobraCommand(t *testing.T, root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	log.Trace().Msgf("Command to execute: same %v", root.CalledAs())

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

// TODO: #233 Replace when we move to go1.18 https://stackoverflow.com/questions/27516387/what-is-the-correct-way-to-find-the-min-between-two-integers-in-go
func Min(a, b int) int {
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

func LoadBadStringsFull() []string {
	file, _ := os.ReadFile("../../testdata/bad_strings_labels.txt")
	return loadBadStrings(file)
}

func LoadBadStringsLabels() []string {
	file, _ := os.ReadFile("../../testdata/bad_strings_labels.txt")
	return loadBadStrings(file)
}

func loadBadStrings(file []byte) []string {
	badStringsRaw := strings.Split(string(file), "\n")
	return FilterStringArray(badStringsRaw, func(s string) bool {
		return !(strings.HasPrefix(s, "#") || strings.HasPrefix(s, "\n"))
	})
}

func FilterStringArray(data []string, f func(string) bool) []string {
	fltd := make([]string, 0)
	for _, e := range data {
		if f(e) {
			fltd = append(fltd, e)
		}
	}
	return fltd
}

func SafeStringStripper(s string) string {
	rChars := SafeCharsRegex()
	return rChars.ReplaceAllString(s, "")
}

func SafeCharsRegex() *regexp.Regexp {
	regexString := "A-Za-z0-9._~!:@,;+-"

	file, _ := os.ReadFile("../../pkg/config/all_emojis.txt")
	emojiArray := strings.Split(string(file), "\n")
	emojiString := strings.Join(emojiArray, "|")

	r := regexp.MustCompile(fmt.Sprintf("[^%s|^%s]", emojiString, regexString))
	return r
}
