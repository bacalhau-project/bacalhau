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

const maxServeTime = 750 * time.Millisecond
const maxTestTime = 10 * time.Second

type ServeSuite struct {
	suite.Suite

	ipfsPort int
	ctx      context.Context
}

func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}

func (s *ServeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	system.InitConfigForTesting(s.T())

	var cancel context.CancelFunc
	s.ctx, cancel = context.WithTimeout(context.Background(), maxTestTime)
	s.T().Cleanup(func() {
		cancel()
	})

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(s.ctx)
	})

	node, err := ipfs.NewLocalNode(s.ctx, cm, []string{})
	s.Require().NoError(err)
	s.ipfsPort = node.APIPort
}

func (s *ServeSuite) serve(extraArgs ...string) int {
	port, err := freeport.GetFreePort()
	s.Require().NoError(err)

	cmd := NewRootCmd()

	// peer set to "none" to avoid accidentally talking to production endpoints
	args := []string{
		"serve",
		"--peer", "none",
		"--ipfs-connect", fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", s.ipfsPort),
		"--api-port", fmt.Sprint(port),
	}
	args = append(args, extraArgs...)

	cmd.SetArgs(args)
	s.T().Logf("Command to execute: %q", args)

	ctx, cancel := context.WithTimeout(s.ctx, maxServeTime)
	s.T().Cleanup(cancel)

	go func() {
		_, err := cmd.ExecuteContextC(ctx)
		s.NoError(err)
	}()

	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.FailNow("Server did not start in time")
		case <-t.C:
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
	port := s.serve()
	healthzText, err := s.curlEndpoint(fmt.Sprintf("http://localhost:%d/healthz", port))
	s.Require().NoError(err)
	var healthzJSON types.HealthInfo
	s.Require().NoError(model.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
	s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
}

func (s *ServeSuite) TestCanSubmitJob() {
	port := s.serve("--node-type", "requester", "--node-type", "compute")
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://localhost:%d", port))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	_, err = client.Submit(s.ctx, job)
	s.NoError(err)
}

func (s *ServeSuite) TestAppliesJobSelectionPolicy() {
	// Networking is disabled by default so we try to submit a networked job and
	// expect it to be rejected.
	port := s.serve("--node-type", "requester")
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://localhost:%d", port))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	job.Spec.Network.Type = model.NetworkHTTP
	job, err = client.Submit(s.ctx, job)
	s.NoError(err)

	state, err := client.GetJobState(s.ctx, job.Metadata.ID)
	s.NoError(err)
	s.Equal(model.JobStateCancelled, state.State, state.State.String())
}
