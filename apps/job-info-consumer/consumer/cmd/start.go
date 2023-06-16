package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bacalhau-project/bacalhau/apps/job-info-consumer/consumer/pkg"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type StartOptions struct {
	postgres    pkg.PostgresDatastoreParams
	swarmPort   int
	peerConnect string
}

func NewStartOptions() *StartOptions {
	return &StartOptions{
		postgres: pkg.PostgresDatastoreParams{
			Host:        getDefaultOptionString("POSTGRES_HOST", "127.0.0.1"), //nolint:gomnd
			Port:        getDefaultOptionInt("POSTGRES_PORT", 5432),           //nolint:gomnd
			Database:    getDefaultOptionString("POSTGRES_DB", "bacalhau"),    //nolint:gomnd
			User:        getDefaultOptionString("POSTGRES_USER", "postgres"),  //nolint:gomnd
			Password:    getDefaultOptionString("POSTGRES_PASSWORD", ""),      //nolint:gomnd
			AutoMigrate: getDefaultOptionBool("POSTGRES_AUTO_MIGRATE", false), //nolint:gomnd
		},
		swarmPort:   getDefaultOptionInt("SWARM_PORT", 1236), //nolint:gomnd
		peerConnect: getDefaultOptionString("BACALHAU_PEER_CONNECT", ""),
	}
}

func newStartCmd() *cobra.Command {
	opts := NewStartOptions()

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the bacalhau job info consumer",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return start(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVar(
		&opts.postgres.Host, "postgres-host", opts.postgres.Host,
		`The host for the postgres server.`,
	)
	cmd.PersistentFlags().IntVar(
		&opts.postgres.Port, "postgres-port", opts.postgres.Port,
		`The port for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.postgres.Database, "postgres-database", opts.postgres.Database,
		`The database for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.postgres.User, "postgres-user", opts.postgres.User,
		`The user for the postgres server.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.postgres.Password, "postgres-password", opts.postgres.Password,
		`The password for the postgres server.`,
	)
	cmd.PersistentFlags().BoolVar(
		&opts.postgres.AutoMigrate, "postgres-auto-migrate", opts.postgres.AutoMigrate,
		`Should auto migrate the database schema.`,
	)
	cmd.PersistentFlags().IntVar(
		&opts.swarmPort, "swarm-port", opts.swarmPort,
		`The port to listen on for swarm connections and GossipSub messages.`,
	)
	cmd.PersistentFlags().StringVar(
		&opts.peerConnect, "peer", opts.peerConnect,
		`The libp2p multiaddress to connect to.`,
	)

	return cmd
}

func start(cmd *cobra.Command, options *StartOptions) error {
	// Cleanup manager ensures that resources are freed before exiting:
	cm := system.NewCleanupManager()
	cm.RegisterCallback(telemetry.Cleanup)
	defer cm.Cleanup(cmd.Context())
	ctx := cmd.Context()

	// Context ensures main goroutine waits until killed with ctrl+c:
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "bacalhau.consumer")
	defer rootSpan.End()

	peers, err := getPeers(options.peerConnect)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Debug().Msgf("libp2p connecting to: %s", peers)

	libp2pHost, err := libp2p.NewHost(options.swarmPort)
	if err != nil {
		return fmt.Errorf("error creating libp2p host: %w", err)
	}

	application, err := pkg.NewApplication(pkg.ApplicationParams{
		PostgresDatastoreParams: options.postgres,
		Libp2pHost:              libp2pHost,
	})
	if err != nil {
		return err
	}
	cm.RegisterCallbackWithContext(application.Stop)

	// Start transport layer
	err = libp2p.ConnectToPeersContinuously(ctx, cm, libp2pHost, peers)
	if err != nil {
		return err
	}

	// Start application
	err = application.Start(ctx)

	log.Info().Msg("Started")
	if err != nil {
		return err
	}
	<-ctx.Done() // block until killed
	return nil
}
