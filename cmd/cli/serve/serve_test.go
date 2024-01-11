//go:build unit || !integration

package serve_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	apitest "github.com/bacalhau-project/bacalhau/pkg/publicapi/test"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"

	"github.com/bacalhau-project/bacalhau/cmd/cli/serve"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	types2 "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/types"
	utilserve "github.com/bacalhau-project/bacalhau/pkg/util/serve"
)

type tServeSuite struct {
	*utilserve.ServeSuite
}

func TestServeSuite(t *testing.T) {
	suite.Run(t, &tServeSuite{ServeSuite: new(utilserve.ServeSuite)})
}

func (s *tServeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	fsRepo := setup.SetupBacalhauRepoForTesting(s.T())
	repoPath, err := fsRepo.Path()
	s.Require().NoError(err)
	s.RepoPath = repoPath

	var cancel context.CancelFunc
	s.Ctx, cancel = context.WithTimeout(context.Background(), utilserve.MaxTestTime)
	s.T().Cleanup(func() {
		cancel()
	})

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(s.Ctx)
	})

	node, err := ipfs.NewNodeWithConfig(s.Ctx, cm, types2.IpfsConfig{PrivateInternal: true})
	s.Require().NoError(err)
	s.IPFSPort = node.APIPort
}

func (s *tServeSuite) serveForCLI(extraArgs ...string) (uint16, error) {
	extraArgs = append(extraArgs, "--repo", s.RepoPath,
		"--peer", serve.DefaultPeerConnect,
		"--private-internal-ipfs")

	// If the slice contains RETURN_ERROR_FLAG, take it out of the array and set returnError to true
	// peer set to "none" to avoid accidentally talking to production endpoints (even though it's default)
	// private-internal-ipfs to avoid accidentally talking to public IPFS nodes (even though it's default)
	returnError, port, err := utilserve.StartServerForTesting(s.ServeSuite, extraArgs)

	if returnError {
		return port, err
	}
	s.NoError(err)
	return port, nil
}

func (s *tServeSuite) TestHealthcheck() {
	port, _ := s.serveForCLI()
	healthzText, statusCode, err := utilserve.CurlEndpoint(s.Ctx, fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", port))
	s.Require().NoError(err)
	var healthzJSON types.HealthInfo
	s.Require().NoError(marshaller.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
	s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *tServeSuite) TestAPIPrintedForComputeNode() {
	port, _ := s.serveForCLI("--node-type", "compute", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/api/v1/compute/debug", port)
	actualUrl := s.Out.String()
	s.Require().Contains(actualUrl, expectedURL)
}

func (s *tServeSuite) TestAPINotPrintedForRequesterNode() {
	port, _ := s.serveForCLI("--node-type", "requester", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/compute/debug", port)
	s.Require().NotContains(s.Out.String(), expectedURL)
}

func (s *tServeSuite) TestCanSubmitJob() {
	docker.MustHaveDocker(s.T())
	ctx := context.Background()
	port, _ := s.serveForCLI("--node-type", "requester", "--node-type", "compute")
	client := client.NewAPIClient("localhost", port)
	clientV2 := clientv2.New(clientv2.Options{
		Address: fmt.Sprintf("http://127.0.0.1:%d", port),
	})
	s.Require().NoError(apitest.WaitForAlive(ctx, clientV2))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	_, err = client.Submit(s.Ctx, job)
	s.NoError(err)
}

func (s *tServeSuite) TestDefaultServeOptionsHavePrivateLocalIpfs() {
	cm := system.NewCleanupManager()

	client, err := serve.SetupIPFSClient(s.Ctx, cm, types2.IpfsConfig{
		Connect:         "",
		PrivateInternal: true,
		SwarmAddresses:  []string{},
	})
	s.Require().NoError(err)

	addrs, err := client.SwarmMultiAddresses(s.Ctx)
	s.Require().NoError(err)

	ip4 := multiaddr.ProtocolWithName("ip4")
	ip6 := multiaddr.ProtocolWithName("ip6")

	for _, addr := range addrs {
		s.T().Logf("Internal IPFS node listening on %s", addr)
		ip, err := addr.ValueForProtocol(ip4.Code)
		if err == nil {
			s.Require().Equal("127.0.0.1", ip)
			continue
		} else {
			s.Require().ErrorIs(err, multiaddr.ErrProtocolNotFound)
		}

		ip, err = addr.ValueForProtocol(ip6.Code)
		if err == nil {
			s.Require().Equal("::1", ip)
			continue
		} else {
			s.Require().ErrorIs(err, multiaddr.ErrProtocolNotFound)
		}
	}

	s.Require().GreaterOrEqual(len(addrs), 1)
}

func (s *tServeSuite) TestGetPeers() {
	// by default it should return no peers
	peers, err := serve.GetPeers(serve.DefaultPeerConnect)
	s.NoError(err)
	s.Require().Equal(0, len(peers))

	// if we set the peer connect to "env" it should return the peers from the env
	for envName, envData := range system.Envs {
		// skip checking environments other than test, because
		// system.GetEnvironment() in getPeers() always returns "test" while testing
		if envName.String() != "test" {
			continue
		}

		// this is required for the below line to succeed as environment is being deprecated.
		config.Set(configenv.Testing)
		peers, err = serve.GetPeers("env")
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
	inputPeers := []string{
		"/ip4/0.0.0.0/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVz", // cspell:disable-line
		"/ip4/0.0.0.0/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcz", // cspell:disable-line
	}
	peerConnect := strings.Join(inputPeers, ",")
	peers, err = serve.GetPeers(peerConnect)
	s.NoError(err)
	s.Require().Equal(inputPeers[0], peers[0].String())
	s.Require().Equal(inputPeers[1], peers[1].String())

	// if we pass invalid multiaddress it should error out
	inputPeers = []string{"foo"}
	peerConnect = strings.Join(inputPeers, ",")
	_, err = serve.GetPeers(peerConnect)
	s.Require().Error(err)
}
