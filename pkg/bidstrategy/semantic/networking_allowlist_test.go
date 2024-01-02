//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

type networkAllowlistTestCase struct {
	Type      models.Network
	Domains   []string
	ShouldBid bool
}

func (tc networkAllowlistTestCase) String() string {
	return fmt.Sprintf(
		"should bid %t with %s networking and domains %s",
		tc.ShouldBid,
		tc.Type,
		strings.Join(tc.Domains, " "),
	)
}

var networkAllowlistTestCases []networkAllowlistTestCase = []networkAllowlistTestCase{
	{models.NetworkNone, []string{}, true},
	{models.NetworkFull, []string{}, false},
	{models.NetworkHTTP, []string{}, true},
	{models.NetworkHTTP, []string{"example.com"}, true},
	{models.NetworkFull, []string{"example.com"}, false},
	{models.NetworkHTTP, []string{"malware.com"}, false},
	{models.NetworkFull, []string{"malware.com"}, false},
	{models.NetworkHTTP, []string{"example.com", "proxy.golang.org"}, true},
	{models.NetworkHTTP, []string{"malware.com", "proxy.golang.org"}, false},
}

func TestNetworkAllowlistStrategyFiltersDomains(t *testing.T) {
	require.NoError(t, exec.Command("jq", "--help").Run(), "Requires `jq` to be installed.")

	strategy := semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
		Command: "../../../ops/terraform/remote_files/scripts/apply-http-allowlist.sh",
	})

	for _, testCase := range networkAllowlistTestCases {
		t.Run(testCase.String(), func(t *testing.T) {
			job := mock.Job()
			job.Task().Network = &models.NetworkConfig{
				Type:    testCase.Type,
				Domains: testCase.Domains,
			}
			resp, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
				Job: *job,
			})

			require.NoError(t, err)
			require.Equal(t, testCase.ShouldBid, resp.ShouldBid, resp.Reason)
		})
	}
}
