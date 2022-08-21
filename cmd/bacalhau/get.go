package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	getLong = templates.LongDesc(i18n.T(`
		Get the results of the job, including stdout and stderr.
`))

	//nolint:lll // Documentation
	getExample = templates.Examples(i18n.T(`
		# Get the results of a job.
		bacalhau get 51225160-807e-48b8-88c9-28311c7899e1

		# Get the results of a job, with a short ID.
		bacalhau get ebd9bf2f
`))

	// Set Defaults (probably a better way to do this)
	OG = NewGetOptions()

	// For the -f flag
)

type GetOptions struct {
	IPFSDownloadSettings ipfs.IPFSDownloadSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		IPFSDownloadSettings: ipfs.IPFSDownloadSettings{
			TimeoutSecs:    600,
			OutputDir:      ".",
			IPFSSwarmAddrs: "",
		},
	}
}

func init() { //nolint:gochecknoinits
	OG.IPFSDownloadSettings.IPFSSwarmAddrs = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	setupDownloadFlags(getCmd, &OG.IPFSDownloadSettings)
}

var getCmd = &cobra.Command{
	Use:     "get [id]",
	Short:   "Get the results of a job",
	Long:    getLong,
	Example: getExample,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		jobID := cmdArgs[0]

		log.Info().Msgf("Fetching results of job '%s'...", jobID)

		j, ok, err := getAPIClient().Get(context.Background(), jobID)

		if !ok {
			cmd.Printf("No job ID found matching ID: %s", jobID)
			return nil
		}

		if err != nil {
			return err
		}

		// todo: deal with jobs with a different verifier
		if j.Spec.Verifier != verifier.VerifierIpfs {
			return fmt.Errorf("job verifier not found: %s", j.Spec.Verifier)
		}

		results, err := getAPIClient().GetResults(context.Background(), j.ID)
		if err != nil {
			return err
		}

		err = ipfs.DownloadJob(
			cm,
			j,
			results,
			OG.IPFSDownloadSettings,
		)

		if err != nil {
			return err
		}

		return nil
	},
}
