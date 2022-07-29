package bacalhau

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	JSONFormat string = "json"
	YAMLFormat string = "yaml"
)

var listOutputFormat string
var tableOutputWide bool
var tableHideHeader bool
var tableMaxJobs int
var tableSortBy ColumnEnum
var tableSortReverse bool
var tableIDFilter string
var tableNoStyle bool

func shortenTime(t time.Time) string { // nolint:unused // Useful function, holding here
	if tableOutputWide {
		return t.Format("06-01-02-15:04:05")
	}

	return t.Format("15:04:05")
}

var DefaultShortenStringLength = 20

func shortenString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < DefaultShortenStringLength {
		return st
	}

	return st[:20] + "..."
}

func shortID(id string) string {
	return id[:8]
}

func getAPIClient() *publicapi.APIClient {
	return publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

func ExecuteTestCobraCommand(t *testing.T, root *cobra.Command, args ...string) (
	c *cobra.Command, output string, err error) { //nolint:unparam // use of t is valuable here
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{})
	root.SetArgs(args)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	log.Trace().Msgf("Command to execute: %v", root.CalledAs())

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

// TODO: #233 Replace when we move to go1.18
// https://stackoverflow.com/questions/27516387/what-is-the-correct-way-to-find-the-min-between-two-integers-in-go
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ReverseList(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

type downloadSettings struct {
	timeoutSecs    int
	outputDir      string
	ipfsSwarmAddrs string
}

func setupDownloadFlags(cmd *cobra.Command, settings downloadSettings) {
	cmd.Flags().IntVar(&settings.timeoutSecs, "timeout-secs",
		settings.timeoutSecs, "Timeout duration for IPFS downloads.")
	cmd.Flags().StringVar(&settings.outputDir, "output-dir",
		settings.outputDir, "Directory to write the output to.")
	cmd.Flags().StringVar(&settings.ipfsSwarmAddrs, "ipfs-swarm-addrs",
		settings.ipfsSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")
}

func downloadJobResults(
	cm *system.CleanupManager,
	resultCIDs []string,
	settings downloadSettings,
) error {
	log.Debug().Msgf("Job has result CIDs: %v", resultCIDs)

	if len(resultCIDs) == 0 {
		log.Info().Msg("Job has no results.")
		return nil
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	log.Debug().Msg("Spinning up IPFS node...")
	n, err := ipfs.NewNode(cm, strings.Split(settings.ipfsSwarmAddrs, ","))
	if err != nil {
		return err
	}

	log.Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client()
	if err != nil {
		return err
	}

	// NOTE: this will run in non-deterministic order
	for _, cid := range resultCIDs {
		outputDir := filepath.Join(settings.outputDir, cid)
		ok, err := system.PathExists(outputDir)
		if err != nil {
			return err
		}
		if ok {
			log.Warn().Msgf("Output directory '%s' already exists, skipping CID '%s'.", outputDir, cid)
			continue
		}

		err = func() error {
			log.Info().Msgf("Downloading result CID '%s' to '%s'...",
				cid, outputDir)

			ctx, cancel := context.WithDeadline(context.Background(),
				time.Now().Add(time.Second*time.Duration(settings.timeoutSecs)))
			defer cancel()

			return cl.Get(ctx, cid, outputDir)
		}()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Error().Msg("Timed out while downloading result.")
			}

			return err
		}
	}

	return nil
}
