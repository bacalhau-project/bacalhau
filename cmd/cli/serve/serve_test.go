//go:build unit || !integration

package serve_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	apitest "github.com/bacalhau-project/bacalhau/pkg/publicapi/test"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"

	cmd2 "github.com/bacalhau-project/bacalhau/cmd/cli"
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
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

const maxServeTime = 15 * time.Second
const maxTestTime = 10 * time.Second
const RETURN_ERROR_FLAG = "RETURN_ERROR"

type ServeSuite struct {
	suite.Suite

	out, err strings.Builder

	ipfsPort int
	ctx      context.Context
	repoPath string
}

func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}

func (s *ServeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	fsRepo := setup.SetupBacalhauRepoForTesting(s.T())
	repoPath, err := fsRepo.Path()
	s.Require().NoError(err)
	s.repoPath = repoPath

	var cancel context.CancelFunc
	s.ctx, cancel = context.WithTimeout(context.Background(), maxTestTime)
	s.T().Cleanup(func() {
		cancel()
	})

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(s.ctx)
	})

	node, err := ipfs.NewNodeWithConfig(s.ctx, cm, types2.IpfsConfig{PrivateInternal: true})
	s.Require().NoError(err)
	s.ipfsPort = node.APIPort
}

func (s *ServeSuite) serve(extraArgs ...string) (uint16, error) {
	// If the slice contains RETURN_ERROR_FLAG, take it out of the array and set returnError to true
	returnError := false
	for i, arg := range extraArgs {
		if arg == RETURN_ERROR_FLAG {
			extraArgs = append(extraArgs[:i], extraArgs[i+1:]...)
			returnError = true
			break
		}
	}

	bigPort, err := freeport.GetFreePort()
	s.Require().NoError(err)
	port := uint16(bigPort)

	cmd := cmd2.NewRootCmd()
	cmd.SetOut(&s.out)
	cmd.SetErr(&s.err)

	// peer set to "none" to avoid accidentally talking to production endpoints (even though it's default)
	// private-internal-ipfs to avoid accidentally talking to public IPFS nodes (even though it's default)
	args := []string{
		"serve",
		"--repo", s.repoPath,
		"--peer", serve.DefaultPeerConnect,
		"--private-internal-ipfs",
		"--port", fmt.Sprint(port),
	}
	args = append(args, extraArgs...)

	cmd.SetArgs(args)
	s.T().Logf("Command to execute: %q", args)

	ctx, cancel := context.WithTimeout(s.ctx, maxServeTime)
	errs, ctx := errgroup.WithContext(ctx)

	s.T().Cleanup(cancel)
	errs.Go(func() error {
		_, err := cmd.ExecuteContextC(ctx)
		if returnError {
			return err
		}
		s.NoError(err)
		return nil
	})

	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			if returnError {
				return 0, errs.Wait()
			}
			s.FailNow("Server did not start in time")
		case <-t.C:
			livezText, statusCode, _ := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/api/v1/livez", port))
			if string(livezText) == "OK" && statusCode == http.StatusOK {
				return port, nil
			}
		}
	}
}

func (s *ServeSuite) curlEndpoint(URL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(s.ctx, "GET", URL, nil)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	defer closer.DrainAndCloseWithLogOnError(s.ctx, "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return responseText, resp.StatusCode, nil
}

func (s *ServeSuite) TestHealthcheck() {
	port, _ := s.serve()
	healthzText, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", port))
	s.Require().NoError(err)
	var healthzJSON types.HealthInfo
	s.Require().NoError(marshaller.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
	s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *ServeSuite) TestAPIPrintedForComputeNode() {
	port, _ := s.serve("--node-type", "compute", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/api/v1/compute/debug", port)
	actualUrl := s.out.String()
	s.Require().Contains(actualUrl, expectedURL)
}

func (s *ServeSuite) TestAPINotPrintedForRequesterNode() {
	port, _ := s.serve("--node-type", "requester", "--log-mode", string(logger.LogModeStation))
	expectedURL := fmt.Sprintf("API: http://0.0.0.0:%d/compute/debug", port)
	s.Require().NotContains(s.out.String(), expectedURL)
}

func (s *ServeSuite) TestCanSubmitJob() {
	docker.MustHaveDocker(s.T())
	port, _ := s.serve("--node-type", "requester", "--node-type", "compute")
	client := client.NewAPIClient("localhost", port)
	clientV2 := clientv2.New(clientv2.Options{
		Address: fmt.Sprintf("http://127.0.0.1:%d", port),
	})
	s.Require().NoError(apitest.WaitForAlive(s.ctx, clientV2))

	job, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	_, err = client.Submit(s.ctx, job)
	s.NoError(err)
}

func (s *ServeSuite) TestDefaultServeOptionsHavePrivateLocalIpfs() {
	cm := system.NewCleanupManager()

	client, err := serve.SetupIPFSClient(s.ctx, cm, types2.IpfsConfig{
		Connect:         "",
		PrivateInternal: true,
		SwarmAddresses:  []string{},
	})
	s.Require().NoError(err)

	addrs, err := client.SwarmMultiAddresses(s.ctx)
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

func (s *ServeSuite) TestGetPeers() {
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
		"/ip4/0.0.0.0/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVz",
		"/ip4/0.0.0.0/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcz",
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

// Begin WebUI Tests
func (s *ServeSuite) Test200ForNotStartingWebUI() {
	port, err := s.serve()
	s.Require().NoError(err, "Error starting server")

	content, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/", port))
	_ = content
	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *ServeSuite) Test200ForRoot() {
	webUIPort, err := freeport.GetFreePort()
	if err != nil {
		s.T().Fatal(err, "Could not get port for web-ui")
	}
	_, err = s.serve("--web-ui", "--web-ui-port", fmt.Sprintf("%d", webUIPort))
	s.Require().NoError(err, "Error starting server")

	_, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/", webUIPort))
	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

// TODO: Can't figure out how to make this test work, it spits out the help text
// func (s *ServeSuite) TestBadBacalhauDir() {
// 	badDirString := "/BADDIR"

// 	// if we set the peer connect to "env" it should return the peers from the env
// 	originalEnv := os.Getenv("BACALHAU_ENVIRONMENT")
// 	defer os.Setenv("BACALHAU_ENVIRONMENT", originalEnv)
// 	os.Setenv("BACALHAU_DIR", badDirString)
// 	_, err := s.serve("--node-type", "requester", "--node-type", "compute", RETURN_ERROR_FLAG)
// 	s.Require().Contains(s.out.String(), "Could not write to")
// 	s.Error(err)
// }
