package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
)

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic

}

var simulatorCmd = &cobra.Command{
	Use:   "simulator",
	Short: "Run the bacalhau simulator",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Cleanup manager ensures that resources are freed before exiting:
		cm := system.NewCleanupManager()
		cm.RegisterCallback(system.CleanupTraceProvider)
		defer cm.Cleanup()
		ctx := cmd.Context()
		localDB, err := inmemory.NewInMemoryDatastore()
		if err != nil {
			return err
		}
		server := simulator.NewServer(ctx, "0.0.0.0", 9075, localDB) //nolint:gomnd
		err = server.ListenAndServe(ctx, cm)
		if err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	},
}
