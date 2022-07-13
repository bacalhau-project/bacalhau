package bacalhau

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	JSONFormat string = "json"
	YAMLFormat string = "yaml"
)

var listOutputFormat string
var tableOutputWide bool
var tableHideHeader bool
var tableMaxJobs int
var tableSortBy ColumnEnum
var tableSortReverse bool
var tableIDFilter string
var tableNoStyle bool

func shortenTime(t time.Time) string { // nolint:unused // Useful function, holding here
	if tableOutputWide {
		return t.Format("06-01-02-15:04:05")
	}

	return t.Format("15:04:05")
}

var DefaultShortenStringLength = 20

func shortenString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < DefaultShortenStringLength {
		return st
	}

	return st[:20] + "..."
}

func shortID(id string) string {
	return id[:8]
}

func getJobResult(job *executor.Job, state *executor.JobState) string {
	if state.ResultsID == "" {
		return "-"
	}
	return "/" + strings.ToLower(job.Spec.Verifier.String()) + "/" + state.ResultsID
}

func getAPIClient() *publicapi.APIClient {
	return publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

func ExecuteTestCobraCommand(t *testing.T, root *cobra.Command, args ...string) (
	c *cobra.Command, output string, err error) { //nolint:unparam // use of t is valuable here
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

	log.Trace().Msgf("Command to execute: %v", root.CalledAs())

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

// TODO: #233 Replace when we move to go1.18
// https://stackoverflow.com/questions/27516387/what-is-the-correct-way-to-find-the-min-between-two-integers-in-go
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

// func RandInt(i int) int {
// 	n, err := rand.Int(rand.Reader, big.NewInt(int64(i)))
// 	if err != nil {
// 		log.Fatal().Msg("could not generate random number")
// 	}

// 	return int(n.Int64())
// }
