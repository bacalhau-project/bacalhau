package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var getDownloadFlags = ipfs.DownloadSettings{
	TimeoutSecs:    10,
	OutputDir:      ".",
	IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
}

func init() { //nolint:gochecknoinits
	setupDownloadFlags(getCmd, &getDownloadFlags)
}

var getCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get the results of a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		jobID := cmdArgs[0]

		log.Info().Msgf("Fetching results of job '%s'...", jobID)

		job, ok, err := getAPIClient().Get(context.Background(), jobID)

		if !ok {
			cmd.Printf("No job ID found matching ID: %s", jobID)
			return nil
		}

		if err != nil {
			return err
		}

		// todo: deal with jobs with a different verifier
		if job.Spec.Verifier != verifier.VerifierIpfs {
			return fmt.Errorf("job verifier not found: %s", job.Spec.Verifier)
		}

		results, err := getAPIClient().GetResults(context.Background(), job.ID)
		if err != nil {
			return err
		}

		err = ipfs.DownloadJob(
			cm,
			job,
			results,
			getDownloadFlags,
		)

		if err != nil {
			return err
		}

		return nil
	},
}
