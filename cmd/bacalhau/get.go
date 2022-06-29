package bacalhau

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var getCmdFlags = struct {
	ipfsURL   string
	outputDir string
}{
	ipfsURL:   "ipfs.io",
	outputDir: ".",
}

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	getCmd.Flags().StringVar(&getCmdFlags.outputDir, "output-dir",
		getCmdFlags.outputDir, "Directory to write the output to.")
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the results of a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error { // nolintunparam // incorrectly suggesting unused
		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		log.Debug().Msgf("Fetching results of job '%s'...", args[0])
		job, ok, err := getAPIClient().Get(context.Background(), args[0])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("job not found")
		}

		var resultCIDs []string
		for _, jobState := range job.State {
			if jobState.ResultsID != "" {
				resultCIDs = append(resultCIDs, jobState.ResultsID)
			}
		}
		log.Debug().Msgf("Job has result CIDs: %v", resultCIDs)

		log.Debug().Msg("Spinning up IPFS client...")
		cl, err := ipfs.NewClient(cm)
		if err != nil {
			return err
		}

		for _, cid := range resultCIDs {
			outputDir := filepath.Join(getCmdFlags.outputDir, cid)
			log.Debug().Msgf("Downloading result CID '%s' to '%s'...",
				cid, outputDir)

			err = cl.Get(context.Background(), cid, outputDir)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
