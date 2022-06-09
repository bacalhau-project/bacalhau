package bacalhau

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/types"
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

func getJobResult(job *types.Job, state *types.JobState) string {
	return "/" + job.Spec.Verifier + "/" + state.ResultsId
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
