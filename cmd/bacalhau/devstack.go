package bacalhau

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/spf13/cobra"
)

func init() {

}

var devstackCmd = &cobra.Command{
	Use:   "devstack",
	Short: "Start a cluster of 3 bacalhau nodes for testing and development",
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx := context.Background()
		ctxWithCancel, cancelFunction := context.WithCancel(ctx)

		_, err := internal.NewDevStack(ctxWithCancel, 3)

		if err != nil {
			cancelFunction()
			return err
		}

		time.Sleep(time.Second * 10)

		fmt.Printf("CLOSING!\n")
		cancelFunction()

		// wait forever because everything else is running in a goroutine
		//select {}

		return nil
	},
}
