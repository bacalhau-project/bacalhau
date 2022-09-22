package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//nolint:gochecknoinits
func init() {
	runCmd.AddCommand(runPythonCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network (see subcommands for supported flavors)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that the server version is compatible with the client version
		serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			log.Err(err)
			return err
		}

		return nil
	},
}
