package runtime

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/testground/sdk-go/ptypes"
)

// RandomTestRunEnv generates a random RunEnv for testing purposes.
func RandomTestRunEnv(t *testing.T) (re *RunEnv, cleanup func()) {
	t.Helper()

	b := make([]byte, 32)
	_, _ = rand.Read(b)

	_, subnet, _ := net.ParseCIDR("127.1.0.1/16")

	odir, err := ioutil.TempDir("", "testground-tests-*")
	if err != nil {
		t.Fatalf("failed to create temp output dir: %s", err)
	}

	rp := RunParams{
		TestPlan:               fmt.Sprintf("testplan-%d", rand.Uint32()),
		TestSidecar:            false,
		TestCase:               fmt.Sprintf("testcase-%d", rand.Uint32()),
		TestRun:                fmt.Sprintf("testrun-%d", rand.Uint32()),
		TestSubnet:             &ptypes.IPNet{IPNet: *subnet},
		TestInstanceCount:      int(1 + (rand.Uint32() % 999)),
		TestInstanceRole:       "",
		TestInstanceParams:     make(map[string]string),
		TestGroupID:            fmt.Sprintf("group-%d", rand.Uint32()),
		TestStartTime:          time.Now(),
		TestGroupInstanceCount: int(1 + (rand.Uint32() % 999)),
		TestOutputsPath:        odir,
		TestDisableMetrics:     false,
	}

	return NewRunEnv(rp), func() {
		_ = os.RemoveAll(odir)
	}
}
