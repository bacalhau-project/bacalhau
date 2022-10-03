package bacalhau

import (
	"fmt"

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
		server := simulator.NewServer(ctx, "0.0.0.0", 9075)
		err := server.ListenAndServe(ctx, cm)
		if err != nil {
			Fatal(fmt.Sprintf("Error starting node: %s", err), 1)
		}
		<-ctx.Done()
		return nil
	},
}
