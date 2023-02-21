package dashboard

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
)

func getCommandLineExecutable() string {
	return os.Args[0]
}

func getDefaultServeOptionString(envName string, defaultValue string) string {
	envValue := os.Getenv(fmt.Sprintf("BACALHAU_DASHBOARD_%s", envName))
	if envValue != "" {
		return envValue
	}
	return defaultValue
}

func getDefaultServeOptionInt(envName string, defaultValue int) int {
	envValue := os.Getenv(fmt.Sprintf("BACALHAU_DASHBOARD_%s", envName))
	if envValue != "" {
		i, err := strconv.Atoi(envValue)
		if err == nil {
			return i
		}
	}
	return defaultValue
}

func FatalErrorHandler(cmd *cobra.Command, msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.Print(msg)
	}
	os.Exit(code)
}

func newModelOptions() model.ModelOptions {
	return model.ModelOptions{
		PostgresHost: getDefaultServeOptionString("POSTGRES_HOST", "127.0.0.1"),
		//nolint:gomnd
		PostgresPort:     getDefaultServeOptionInt("POSTGRES_PORT", 5432),
		PostgresDatabase: getDefaultServeOptionString("POSTGRES_DATABASE", "bacalhau"),
		PostgresUser:     getDefaultServeOptionString("POSTGRES_USER", ""),
		PostgresPassword: getDefaultServeOptionString("POSTGRES_PASSWORD", ""),
	}
}

func setupModelOptions(cmd *cobra.Command, opts *model.ModelOptions) {
	cmd.PersistentFlags().StringVar(
		&opts.PostgresHost, "postgres-host", opts.PostgresHost,
		`The host for the postgres server.`,
	)
	cmd.PersistentFlags().IntVar(
		&opts.PostgresPort, "postgres-port", opts.PostgresPort,
		`The port for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.PostgresDatabase, "postgres-database", opts.PostgresDatabase,
		`The database for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.PostgresUser, "postgres-user", opts.PostgresUser,
		`The user for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.PostgresPassword, "postgres-password", opts.PostgresPassword,
		`The password for the postgres server.`,
	)
}

func getPeers(peerConnect string) ([]multiaddr.Multiaddr, error) {
	var peersStrings []string
	if peerConnect == "none" {
		peersStrings = []string{}
	} else if peerConnect == "" {
		peersStrings = system.Envs[system.EnvironmentProd].BootstrapAddresses
	} else {
		peersStrings = strings.Split(peerConnect, ",")
	}

	peers := make([]multiaddr.Multiaddr, 0, len(peersStrings))
	for _, peer := range peersStrings {
		parsed, err := multiaddr.NewMultiaddr(peer)
		if err != nil {
			return nil, err
		}
		peers = append(peers, parsed)
	}
	return peers, nil
}
