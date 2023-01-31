package dashboard

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/server"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	serveLong = templates.LongDesc(i18n.T(`
		Start the bacalhau dashboard server.
		`))

	serveExample = templates.Examples(i18n.T(`
		TBD`))
)

type ServeOptions struct {
	ServerOptions server.ServerOptions
	ModelOptions  model.ModelOptions
}

func NewServeOptions() *ServeOptions {
	return &ServeOptions{
		ServerOptions: server.ServerOptions{
			Host: getDefaultServeOptionString("HOST", "0.0.0.0"),
			//nolint:gomnd
			Port:      getDefaultServeOptionInt("PORT", 80),
			JWTSecret: getDefaultServeOptionString("JWT_SECRET", ""),
		},
		ModelOptions: newModelOptions(),
	}
}

func newServeCmd() *cobra.Command {
	serveOptions := NewServeOptions()

	serveCmd := &cobra.Command{
		Use:     "serve",
		Short:   "Start the bacalhau dashboard server",
		Long:    serveLong,
		Example: serveExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return serve(cmd, serveOptions)
		},
	}

	serveCmd.PersistentFlags().StringVar(
		&serveOptions.ServerOptions.Host, "host", serveOptions.ServerOptions.Host,
		`The host to bind the dashboard server to.`,
	)
	serveCmd.PersistentFlags().IntVar(
		&serveOptions.ServerOptions.Port, "port", serveOptions.ServerOptions.Port,
		`The host to bind the dashboard server to.`,
	)
	serveCmd.PersistentFlags().StringVar(
		&serveOptions.ServerOptions.JWTSecret, "jwt-secret", serveOptions.ServerOptions.JWTSecret,
		`The signing secret we use for JWT tokens.`,
	)

	setupModelOptions(serveCmd, &serveOptions.ModelOptions)

	return serveCmd
}

func serve(cmd *cobra.Command, options *ServeOptions) error {
	// Cleanup manager ensures that resources are freed before exiting:
	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTraceProvider)
	defer cm.Cleanup()
	ctx := cmd.Context()

	if options.ServerOptions.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// Context ensures main goroutine waits until killed with ctrl+c:
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "dashboard/api/cmd/dashboard/serve")
	defer rootSpan.End()

	model, err := model.NewModelAPI(options.ModelOptions)
	if err != nil {
		return err
	}

	model.Start(ctx)

	server, err := server.NewServer(
		options.ServerOptions,
		model,
	)
	if err != nil {
		return err
	}

	go func() {
		err := server.ListenAndServe(ctx, cm)
		if err != nil {
			panic(err)
		}
	}()

	log.Info().Msgf("Dashboard server listening on %s:%d", options.ServerOptions.Host, options.ServerOptions.Port)

	<-ctx.Done() // block until killed
	return nil
}
