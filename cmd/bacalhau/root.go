package bacalhau

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
)

var jsonrpcPort int
var jsonrpcHost string
var developmentMode bool

func init() {
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(submitCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(resultsCmd)
	RootCmd.AddCommand(devstackCmd)
	RootCmd.PersistentFlags().IntVar(
		&jsonrpcPort, "jsonrpc-port", 1234,
		`The port for the client and server to communicate on (via jsonrpc).`,
	)
	RootCmd.PersistentFlags().StringVar(
		&jsonrpcHost, "jsonrpc-host", "0.0.0.0",
		`The port for the client and server to communicate on (via jsonrpc).`,
	)
	RootCmd.PersistentFlags().BoolVar(
		&developmentMode, "dev", false,
		`Development mode makes it easier to run multiple bacalhau nodes on the same machine.`,
	)
}

var RootCmd = &cobra.Command{
	Use:   "bacalhau",
	Short: "Compute over data",
	Long:  `Compute over data`,
}

func Execute(version string, ctx context.Context) {

	_, span := otel.Tracer("bacalhau.org").Start(ctx, "Root Span")
	defer span.End()

	RootCmd.Version = version
	setVersion()

	if err := RootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		ctx.Done()
		span.End()
		os.Exit(1)
	}

	ctx.Done()
	span.End()
}

func setVersion() {
	template := fmt.Sprintf("Bacalhau Version: %s\n", RootCmd.Version)
	RootCmd.SetVersionTemplate(template)
}
