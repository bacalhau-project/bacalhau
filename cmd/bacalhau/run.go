package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/spf13/cobra"
)

//nolint:gochecknoinits
func init() {
	runCmd.AddCommand(runPythonCmd)
}

var runCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run a job on the network (see subcommands for supported flavors)",
	PreRun: applyPorcelainLogLevel,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Check that the server version is compatible with the client version
		serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			Fatal(fmt.Sprintf("version validation failed: %s", err), 1)
			return err
		}

		return nil
	},
}
