package main

import (
	"github.com/spf13/cobra"
)

var jsonrpcPort int

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.PersistentFlags().IntVar(
		&jsonrpcPort, "jsonrpc-port", 1234,
		`The port for the client and server to communicate on over localhost (via jsonrpc).`,
	)
}

var rootCmd = &cobra.Command{
	Use:   "bacalhau",
	Short: "Compute over data",
	Long:  `Compute over data`,
}
