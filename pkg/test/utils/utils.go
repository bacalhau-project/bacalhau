package testutils

import (
	"context"
	"regexp"
	"testing"

	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

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
func MustHaveIPFS(t testing.TB, ipfsConnect string) {
	if !IsIPFSEnabled(ipfsConnect) {
		t.Skip("Cannot run this test because it IPFS Connect is not configured")
	}
}

// IsIPFSEnabled will return true if the test is running in an environment that can support IPFS.
func IsIPFSEnabled(ipfsConnect string) bool {
	return ipfsConnect != ""
}

// StartNats will start a NATS server on a random port and return a server and client instances
func StartNats(t *testing.T) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	port, err := network.GetFreePort()
	require.NoError(t, err)

	opts := &natstest.DefaultTestOptions
	opts.Port = port

	natsServer := natstest.RunServer(opts)
	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	return natsServer, nc
}
