//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServeSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}

// Before each test
func (suite *ServeSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), system.InitConfigForTesting(suite.T()))
	suite.rootCmd = RootCmd
}

func writeToServeChannel(rootCmd *cobra.Command, port int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Starting")

	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	ipfsPort, _ := freeport.GetFreePort()

	args := []string{"serve", "--ipfs-connect", fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", ipfsPort), "--api-port", fmt.Sprintf("%d", port)}

	rootCmd.SetArgs(args)

	log.Trace().Msgf("Command to execute: %v", rootCmd.CalledAs())

	_, _ = rootCmd.ExecuteC()
}

func curlEndpoint(URL string) (string, error) {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer closer.DrainAndCloseWithLogOnError(context.Background(), "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(responseText), nil
}

func (suite *ServeSuite) TestRun_GenericServe() {

	*OS = *NewServeOptions()
	OS.PeerConnect = "none" // avoid accidentally talking to production endpoints

	port, err := freeport.GetFreePort()

	require.NoError(suite.T(), err, "Error getting free port.")

	var wg sync.WaitGroup
	wg.Add(2)

	go writeToServeChannel(suite.rootCmd, port, &wg)

	timeoutInMilliseconds := 20 * 1000
	currentTime := 0
	for {
		time.Sleep(100 * time.Millisecond)
		currentTime = currentTime + 100
		livezText, _ := curlEndpoint(fmt.Sprintf("http://localhost:%d/livez", port))
		if livezText == "OK" {
			healthzText, _ := curlEndpoint(fmt.Sprintf("http://localhost:%d/healthz", port))
			healthzJSON := &types.HealthInfo{}
			err := model.JSONUnmarshalWithMax([]byte(healthzText), healthzJSON)
			require.NoError(suite.T(), err, "Error unmarshalling healthz JSON.")
			require.Greater(suite.T(), int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
			wg.Done()
			break
		}

		if currentTime > timeoutInMilliseconds {
			require.Fail(suite.T(), fmt.Sprintf("Server did not start in %d", timeoutInMilliseconds))
			wg.Done()
			break
		}
	}

}
