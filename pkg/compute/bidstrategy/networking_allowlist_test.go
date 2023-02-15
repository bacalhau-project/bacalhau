//go:build unit || !integration

package bidstrategy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

type networkAllowlistTestCase struct {
	Type      model.Network
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
	{model.NetworkNone, []string{}, true},
	{model.NetworkFull, []string{}, false},
	{model.NetworkHTTP, []string{}, true},
	{model.NetworkHTTP, []string{"example.com"}, true},
	{model.NetworkFull, []string{"example.com"}, false},
	{model.NetworkHTTP, []string{"malware.com"}, false},
	{model.NetworkFull, []string{"malware.com"}, false},
	{model.NetworkHTTP, []string{"example.com", "proxy.golang.org"}, true},
	{model.NetworkHTTP, []string{"malware.com", "proxy.golang.org"}, false},
}

func TestNetworkAllowlistStrategyFiltersDomains(t *testing.T) {
	require.NoError(t, exec.Command("jq", "--help").Run(), "Requires `jq` to be installed.")

	strategy := NewExternalCommandStrategy(ExternalCommandStrategyParams{
		Command: "../../../ops/terraform/remote_files/scripts/apply-http-allowlist.sh",
	})

	for _, testCase := range networkAllowlistTestCases {
		t.Run(testCase.String(), func(t *testing.T) {
			resp, err := strategy.ShouldBid(context.Background(), BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Network: model.NetworkConfig{
							Type:    testCase.Type,
							Domains: testCase.Domains,
						},
					},
				},
			})

			require.NoError(t, err)
			require.Equal(t, testCase.ShouldBid, resp.ShouldBid, resp.Reason)
		})
	}
}
