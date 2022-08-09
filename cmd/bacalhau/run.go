package bacalhau

import (
	"fmt"

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
		serverVersion, _ := getAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			err = fmt.Errorf("version mismatch, please upgrade your client: %s", err)
			log.Err(err)
			return err
		}

		return nil
	},
}
