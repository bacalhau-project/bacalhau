package testutils

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func GetJobFromTestOutput(ctx context.Context, t *testing.T, c clientv2.API, out string) *models.Job {
	jobID := system.FindJobIDInTestOutput(out)
	uuidRegex := regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	require.Regexp(t, uuidRegex, jobID, "Job ID should be a UUID")

	j, err := c.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobID: jobID,
	})
	require.NoError(t, err)
	require.NotNil(t, j, "Failed to get job with ID: %s", out)
	return j.Job
}

// MustHaveIPFS will skip the test if the test is running in an environment that cannot support IPFS.
// Otherwise it returns an IPFS connect string
func MustHaveIPFS(t testing.TB, cfg types.Bacalhau) {
	downloaderConfigured := cfg.ResultDownloaders.IsNotDisabled(models.StorageSourceIPFS) &&
		cfg.ResultDownloaders.Types.IPFS.Endpoint != ""
	inputSourceConfigured := cfg.InputSources.IsNotDisabled(models.StorageSourceIPFS) &&
		cfg.InputSources.Types.IPFS.Endpoint != ""
	publisherConfigured := cfg.Publishers.IsNotDisabled(models.PublisherIPFS) &&
		cfg.Publishers.Types.IPFS.Endpoint != ""

	if !(downloaderConfigured && inputSourceConfigured && publisherConfigured) {
		t.Skip("Cannot run this test because it IPFS Connect is not configured")
	}
}

// IsIPFSEnabled will return true if the test is running in an environment that can support IPFS.
func IsIPFSEnabled(ipfsConnect string) bool {
	return ipfsConnect != ""
}

// startNatsOnPort will start a NATS server on a specific port and return a server and client instances
func startNatsOnPort(t *testing.T, port int) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	opts := &natstest.DefaultTestOptions
	opts.Port = port

	natsServer := natstest.RunServer(opts)
	nc, err := nats.Connect(natsServer.ClientURL(),
		nats.ReconnectBufSize(-1),                 // disable reconnect buffer so client fails fast if disconnected
		nats.ReconnectWait(200*time.Millisecond),  //nolint:mnd // reduce reconnect wait to fail fast in tests
		nats.FlusherTimeout(100*time.Millisecond), //nolint:mnd // reduce flusher timeout to speed up tests
	)
	require.NoError(t, err)
	return natsServer, nc
}

// StartNats will start a NATS server on a random port and return a server and client instances
func StartNats(t *testing.T) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	port, err := network.GetFreePort()
	require.NoError(t, err)

	return startNatsOnPort(t, port)
}

// RestartNatsServer will restart the NATS server and return a new server and client using the same port
func RestartNatsServer(t *testing.T, natsServer *natsserver.Server) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	natsServer.Shutdown()

	u, err := url.Parse(natsServer.ClientURL())
	require.NoError(t, err, "Failed to parse NATS server URL %s", natsServer.ClientURL())

	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err, "Failed to convert port %s to int", u.Port())

	return startNatsOnPort(t, port)
}
