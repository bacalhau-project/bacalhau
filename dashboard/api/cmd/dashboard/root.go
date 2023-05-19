package dashboard

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
)

var Fatal = FatalErrorHandler

func NewRootCmd() *cobra.Command {
	RootCmd := &cobra.Command{
		Use:   getCommandLineExecutable(),
		Short: "Dashboard",
		Long:  `Dashboard`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.ConfigureLogging(logger.LogModeDefault)
		},
	}
	RootCmd.AddCommand(newServeCmd())
	RootCmd.AddCommand(newUserCmd())
	RootCmd.AddCommand(newImportCommand())
	return RootCmd
}

func Execute() {
	RootCmd := NewRootCmd()
	RootCmd.SetContext(context.Background())
	RootCmd.SetOutput(system.Stdout)
	if err := RootCmd.Execute(); err != nil {
		Fatal(RootCmd, err.Error(), 1)
	}
}
