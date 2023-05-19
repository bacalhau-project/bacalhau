//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

const maxServeTime = 1 * time.Second
const maxTestTime = 10 * time.Second

type ServeSuite struct {
	suite.Suite

	out, err strings.Builder

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

func (s *ServeSuite) serve(extraArgs ...string) uint16 {
	bigPort, err := freeport.GetFreePort()
	s.Require().NoError(err)
	port := uint16(bigPort)

	cmd := NewRootCmd()
	cmd.SetOut(&s.out)
	cmd.SetErr(&s.err)

	// peer set to "none" to avoid accidentally talking to production endpoints (even though it's default)
	// private-internal-ipfs to avoid accidentally talking to public IPFS nodes (even though it's default)
	args := []string{
		"serve",
		"--peer", DefaultPeerConnect,
		"--private-internal-ipfs",
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

func (s *ServeSuite) TestAPIPrintedForComputeNode() {
	port := s.serve("--node-type", "compute", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/compute/debug", port)
	s.Require().Contains(s.out.String(), expectedURL)
}

func (s *ServeSuite) TestAPINotPrintedForRequesterNode() {
	port := s.serve("--node-type", "requester", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/compute/debug", port)
	s.Require().NotContains(s.out.String(), expectedURL)
}

func (s *ServeSuite) TestCanSubmitJob() {
	port := s.serve("--node-type", "requester", "--node-type", "compute")
	client := publicapi.NewRequesterAPIClient("localhost", port)

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	_, err = client.Submit(s.ctx, job)
	s.NoError(err)
}

func (s *ServeSuite) TestAppliesJobSelectionPolicy() {
	// Networking is disabled by default so we try to submit a networked job and
	// expect it to be rejected.
	port := s.serve("--node-type", "requester")
	client := publicapi.NewRequesterAPIClient("localhost", port)

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	job.Spec.Network.Type = model.NetworkHTTP
	job, err = client.Submit(s.ctx, job)
	s.NoError(err)

	state, err := client.GetJobState(s.ctx, job.Metadata.ID)
	s.NoError(err)
	s.Equal(model.JobStateCancelled, state.State, state.State.String())
}

func (s *ServeSuite) TestDefaultServeOptionsConnectToLocalIpfs() {
	cm := system.NewCleanupManager()
	OS := NewServeOptions()

	client, err := ipfsClient(s.ctx, OS, cm)
	s.Require().NoError(err)

	swarmAddresses, err := client.SwarmAddresses(s.ctx)
	s.NoError(err)
	// an IPFS local node usually returns 2 addresses
	s.Require().Equal(2, len(swarmAddresses))
}

func (s *ServeSuite) TestGetPeers() {
	// by default it should return no peers
	OS := NewServeOptions()
	peers, err := getPeers(OS)
	s.NoError(err)
	s.Require().Equal(0, len(peers))

	// if we set the peer connect to "env" it should return the peers from the env
	originalEnv := os.Getenv("BACALHAU_ENVIRONMENT")
	defer os.Setenv("BACALHAU_ENVIRONMENT", originalEnv)
	for envName, envData := range system.Envs {
		// skip checking environments other than test, because
		// system.GetEnvironment() in getPeers() always returns "test" while testing
		if envName.String() != "test" {
			continue
		}

		OS = NewServeOptions()
		OS.PeerConnect = "env"
		peers, err = getPeers(OS)
		s.NoError(err)
		s.Require().NotEmpty(peers, "getPeers() returned an empty slice")
		// search each peer in env BootstrapAddresses
		for _, peer := range peers {
			found := false
			for _, envPeer := range envData.BootstrapAddresses {
				if peer.String() == envPeer {
					found = true
					break
				}
			}
			s.Require().True(found, "Peer %s not found in env %s", peer, envName)
		}
	}

	// if we pass multiaddresses it should just return them
	OS = NewServeOptions()
	inputPeers := []string{
		"/ip4/0.0.0.0/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVz",
		"/ip4/0.0.0.0/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcz",
	}
	OS.PeerConnect = strings.Join(inputPeers, ",")
	peers, err = getPeers(OS)
	s.NoError(err)
	s.Require().Equal(inputPeers[0], peers[0].String())
	s.Require().Equal(inputPeers[1], peers[1].String())

	// if we pass invalid multiaddress it should error out
	OS = NewServeOptions()
	inputPeers = []string{"foo"}
	OS.PeerConnect = strings.Join(inputPeers, ",")
	_, err = getPeers(OS)
	s.Require().Error(err)
}
