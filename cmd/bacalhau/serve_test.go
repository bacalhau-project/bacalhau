//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
)

const maxTestTime time.Duration = 750 * time.Millisecond

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServeSuite struct {
	suite.Suite

	ipfsPort int
	ctx      context.Context
	cancel   context.CancelFunc
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

	s.ctx, s.cancel = context.WithTimeout(context.Background(), maxTestTime)

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(context.Background())
	})

	node, err := ipfs.NewLocalNode(s.ctx, cm, []string{})
	s.NoError(err)
	s.ipfsPort = node.APIPort
}

func (s *ServeSuite) TearDownTest() {
	s.cancel()
}

func (s *ServeSuite) Serve(extraArgs ...string) int {
	port, err := freeport.GetFreePort()
	s.NoError(err)

	cmd := NewRootCmd()

	// peer set to none to avoid accidentally talking to production endpoints
	args := []string{
		"serve",
		"--peer", "none",
		"--ipfs-connect", fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", s.ipfsPort),
		"--api-port", fmt.Sprint(port),
	}
	args = append(args, extraArgs...)

	cmd.SetArgs(args)
	s.T().Logf("Command to execute: %q", cmd.CalledAs())

	go func() {
		_, err := cmd.ExecuteContextC(s.ctx)
		s.NoError(err)
	}()

	for {
		select {
		case <-s.ctx.Done():
			s.FailNow("Server did not start in time")
		default:
			livezText, _ := s.curlEndpoint(fmt.Sprintf("http://localhost:%d/livez", port))
			if string(livezText) == "OK" {
				return port
			}
		}
	}
}

func (s *ServeSuite) curlEndpoint(URL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(s.ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closer.DrainAndCloseWithLogOnError(s.ctx, "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return responseText, nil
}

func (s *ServeSuite) TestHealthcheck() {
	port := s.Serve()
	healthzText, err := s.curlEndpoint(fmt.Sprintf("http://localhost:%d/healthz", port))
	s.NoError(err)
	var healthzJSON types.HealthInfo
	s.Require().NoError(model.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
	s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
}

func (s *ServeSuite) TestCanSubmitJob() {
	port := s.Serve("--node-type", "requester", "--node-type", "compute")
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://localhost:%d", port))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.NoError(err)

	_, err = client.Submit(s.ctx, job)
	s.NoError(err)
}

func (s *ServeSuite) TestAppliesJobSelectionPolicy() {
	// Networking is disabled by default so we try to submit a networked job and
	// expect it to be rejected.
	port := s.Serve("--node-type", "requester")
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://localhost:%d", port))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.NoError(err)

	job.Spec.Network.Type = model.NetworkHTTP
	_, err = client.Submit(s.ctx, job)
	s.ErrorContains(err, "job is unacceptable")
}
