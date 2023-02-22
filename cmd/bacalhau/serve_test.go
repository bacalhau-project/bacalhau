//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServeSuite struct {
	suite.Suite

	ipfsPort int
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}

// Before each test
func (s *ServeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	s.Require().NoError(system.InitConfigForTesting(s.T()))

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(context.Background())
	})

	node, err := ipfs.NewLocalNode(context.Background(), cm, []string{})
	s.NoError(err)
	s.ipfsPort = node.APIPort
}

func (s *ServeSuite) writeToServeChannel(rootCmd *cobra.Command, port int) {
	s.T().Log("Starting")

	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	// peer set to none to avoid accidentally talking to production endpoints
	args := []string{
		"serve",
		"--peer", "none",
		"--ipfs-connect", fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", s.ipfsPort),
		"--api-port", fmt.Sprintf("%d", port),
	}

	rootCmd.SetArgs(args)

	log.Trace().Msgf("Command to execute: %v", rootCmd.CalledAs())

	_, err := rootCmd.ExecuteC()
	s.Require().NoError(err)
}

func curlEndpoint(URL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseText, nil
}

func (s *ServeSuite) TestRun_GenericServe() {
	port, err := freeport.GetFreePort()
	s.Require().NoError(err, "Error getting free port.")

	go s.writeToServeChannel(NewRootCmd(), port)

	timeout := time.NewTicker(20 * time.Second)
	defer timeout.Stop()

	for {
		time.Sleep(100 * time.Millisecond)
		select {
		case <-timeout.C:
			s.Require().Fail("Server did not start in time")
		default:
			livezText, _ := curlEndpoint(fmt.Sprintf("http://localhost:%d/livez", port))
			if string(livezText) == "OK" {
				healthzText, err := curlEndpoint(fmt.Sprintf("http://localhost:%d/healthz", port))
				s.NoError(err)
				var healthzJSON types.HealthInfo
				s.Require().NoError(model.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
				s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
				return
			}
		}
	}
}
