package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

var listOutputFormat string
var tableOutputWide bool

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
	return publicapi.NewAPIClient(fmt.Sprintf("%s:%d", apiHost, apiPort))
}
