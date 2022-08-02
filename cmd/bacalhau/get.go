package bacalhau

import (
	"context"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var getDownloadFlags = ipfs.DownloadSettings{
	TimeoutSecs:    10,
	OutputDir:      ".",
	IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
}

func init() { // nolint:gochecknoinits
	setupDownloadFlags(getCmd, getDownloadFlags)
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the results of a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		jobID := args[0]

		log.Info().Msgf("Fetching results of job '%s'...", jobID)
		resolver := getAPIClient().GetJobStateResolver()
		resultCIDs, err := resolver.GetResults(context.Background(), jobID)
		if err != nil {
			return err
		}

		err = ipfs.DownloadCIDs(
			cm,
			resultCIDs,
			getDownloadFlags,
		)
		if err != nil {
			return err
		}

		return nil
	},
}
