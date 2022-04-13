package bacalhau

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/ipfs"

	"github.com/spf13/cobra"
)

var devStackBadActors int

func init() {
	devstackCmd.PersistentFlags().IntVar(
		&devStackBadActors, "bad-actors", 0,
		`How many nodes should be bad actors`,
	)
}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of 3 bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error { // nolint
		result, err := ipfs.IpfsCommand("", []string{"version"})

		ctx := cmd.Context()
		tracer := otel.GetTracerProvider().Tracer("bacalhau.org") // if not already in scope
		_, span := tracer.Start(ctx, "DevStack Span")

		// In LIFO order
		defer span.End()

		if err != nil {
			log.Error().Msg(fmt.Sprintf("Error running command 'ipfs version': %s", err))
			return err
		}

		if strings.Contains(result, "0.12.0") {
			err = fmt.Errorf("\n********************\nDue to a regression, we do not support 0.12.0. Please install from here:\nhttps://ipfs.io/ipns/dist.ipfs.io/go-ipfs/v0.11.0/go-ipfs_v0.11.0_linux-amd64.tar.gz\n********************\n")
			log.Error().Err(err)
			return err
		}

		stack, err := internal.NewDevStack(ctx, 3, devStackBadActors)

		if err != nil {
			return err
		}

		stack.PrintNodeInfo()

		nodeChannel := make(chan os.Signal, 1)
		signal.Notify(nodeChannel, syscall.SIGINT, syscall.SIGTERM)
		done := make(chan bool, 1)

		go func() {

			for range nodeChannel {
				// need some time to let ipfs processes shut down
				span.End()
				time.Sleep(time.Second * 2)
				log.Info().Msgf("Force quit.")
				done <- true
			}
		}()
		<-done

		return nil
	},
}
