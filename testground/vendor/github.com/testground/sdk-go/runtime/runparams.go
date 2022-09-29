package runtime

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/testground/sdk-go/ptypes"
)

// RunParams encapsulates the runtime parameters for this test.
type RunParams struct {
	TestPlan string `json:"plan"`
	TestCase string `json:"case"`
	TestRun  string `json:"run"`

	TestRepo   string `json:"repo,omitempty"`
	TestCommit string `json:"commit,omitempty"`
	TestBranch string `json:"branch,omitempty"`
	TestTag    string `json:"tag,omitempty"`

	TestOutputsPath string `json:"outputs_path,omitempty"`
	TestTempPath    string `json:"temp_path,omitempty"`

	TestInstanceCount  int               `json:"instances"`
	TestInstanceRole   string            `json:"role,omitempty"`
	TestInstanceParams map[string]string `json:"params,omitempty"`

	TestGroupID            string `json:"group,omitempty"`
	TestGroupInstanceCount int    `json:"group_instances,omitempty"`

	// true if the test has access to the sidecar.
	TestSidecar bool `json:"test_sidecar,omitempty"`

	// The subnet on which this test is running.
	//
	// The test instance can use this to pick an IP address and/or determine
	// the "data" network interface.
	//
	// This will be 127.1.0.0/16 when using the local exec runner.
	TestSubnet    *ptypes.IPNet `json:"network,omitempty"`
	TestStartTime time.Time     `json:"start_time,omitempty"`

	// TestCaptureProfiles lists the profile types to capture. These are
	// SDK-dependent. The Go SDK supports these profiles:
	//
	// * cpu => value ignored; CPU profile spans the entire life of the test.
	// * any supported profile type https://golang.org/pkg/runtime/pprof/#Profile =>
	//   value is a string representation of time.Duration, referring to
	//   the frequency at which profiles will be captured.
	TestCaptureProfiles map[string]string `json:"capture_profiles,omitempty"`

	// TestDisableMetrics disables Influx batching. It is false by default.
	TestDisableMetrics bool `json:"disable_metrics,omitempty"`
}

// ParseRunParams parses a list of environment variables into a RunParams.
func ParseRunParams(env []string) (*RunParams, error) {
	m, err := ParseKeyValues(env)
	if err != nil {
		return nil, err
	}

	return &RunParams{
		TestBranch:             m[EnvTestBranch],
		TestCase:               m[EnvTestCase],
		TestGroupID:            m[EnvTestGroupID],
		TestGroupInstanceCount: toInt(m[EnvTestGroupInstanceCount]),
		TestInstanceCount:      toInt(m[EnvTestInstanceCount]),
		TestInstanceParams:     unpackParams(m[EnvTestInstanceParams]),
		TestInstanceRole:       m[EnvTestInstanceRole],
		TestOutputsPath:        m[EnvTestOutputsPath],
		TestTempPath:           m[EnvTestTempPath],
		TestPlan:               m[EnvTestPlan],
		TestRepo:               m[EnvTestRepo],
		TestRun:                m[EnvTestRun],
		TestSidecar:            toBool(m[EnvTestSidecar]),
		TestStartTime:          toTime(EnvTestStartTime),
		TestSubnet:             toNet(m[EnvTestSubnet]),
		TestTag:                m[EnvTestTag],
		TestCaptureProfiles:    unpackParams(m[EnvTestCaptureProfiles]),
		TestDisableMetrics:     toBool(m[EnvTestDisableMetrics]),
	}, nil
}

func (rp *RunParams) ToEnvVars() map[string]string {
	packParams := func(in map[string]string) string {
		if in == nil {
			return ""
		}
		arr := make([]string, 0, len(in))
		for k, v := range in {
			arr = append(arr, k+"="+v)
		}
		return strings.Join(arr, "|")
	}

	out := map[string]string{
		EnvTestBranch:             rp.TestBranch,
		EnvTestCase:               rp.TestCase,
		EnvTestGroupID:            rp.TestGroupID,
		EnvTestGroupInstanceCount: strconv.Itoa(rp.TestGroupInstanceCount),
		EnvTestInstanceCount:      strconv.Itoa(rp.TestInstanceCount),
		EnvTestInstanceParams:     packParams(rp.TestInstanceParams),
		EnvTestInstanceRole:       rp.TestInstanceRole,
		EnvTestOutputsPath:        rp.TestOutputsPath,
		EnvTestTempPath:           rp.TestTempPath,
		EnvTestPlan:               rp.TestPlan,
		EnvTestRepo:               rp.TestRepo,
		EnvTestRun:                rp.TestRun,
		EnvTestSidecar:            strconv.FormatBool(rp.TestSidecar),
		EnvTestStartTime:          rp.TestStartTime.Format(time.RFC3339),
		EnvTestSubnet:             rp.TestSubnet.String(),
		EnvTestTag:                rp.TestTag,
		EnvTestCaptureProfiles:    packParams(rp.TestCaptureProfiles),
		EnvTestDisableMetrics:     strconv.FormatBool(rp.TestDisableMetrics),
	}

	return out
}

// IsParamSet checks if a certain parameter is set.
func (rp *RunParams) IsParamSet(name string) bool {
	_, ok := rp.TestInstanceParams[name]
	return ok
}

// StringParam returns a string parameter, or "" if the parameter is not set.
func (rp *RunParams) StringParam(name string) string {
	v, ok := rp.TestInstanceParams[name]
	if !ok {
		panic(fmt.Errorf("%s was not set", name))
	}
	return v
}

func (rp *RunParams) SizeParam(name string) uint64 {
	v := rp.TestInstanceParams[name]
	m, err := humanize.ParseBytes(v)
	if err != nil {
		panic(err)
	}
	return m
}

// IntParam returns an int parameter, or -1 if the parameter is not set or
// the conversion failed. It panics on error.
func (rp *RunParams) IntParam(name string) int {
	v, ok := rp.TestInstanceParams[name]
	if !ok {
		panic(fmt.Errorf("%s was not set", name))
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return i
}

// FloatParam returns a float64 parameter, or -1.0 if the parameter is not set or
// the conversion failed. It panics on error.
func (rp *RunParams) FloatParam(name string) float64 {
	v, ok := rp.TestInstanceParams[name]
	if !ok {
		return -1.0
	}

	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		panic(err)
	}
	return f
}

// BooleanParam returns the Boolean value of the parameter, or false if not passed
func (rp *RunParams) BooleanParam(name string) bool {
	s, ok := rp.TestInstanceParams[name]
	return ok && strings.ToLower(s) == "true"
}

// StringArrayParam returns an array of string parameter, or an empty array
// if it does not exist. It panics on error.
func (rp *RunParams) StringArrayParam(name string) []string {
	var a []string
	rp.JSONParam(name, &a)
	return a
}

// SizeArrayParam returns an array of uint64 elements which represent sizes,
// in bytes. If the response is nil, then there was an error parsing the input.
// It panics on error.
func (rp *RunParams) SizeArrayParam(name string) []uint64 {
	humanSizes := rp.StringArrayParam(name)
	var sizes []uint64

	for _, size := range humanSizes {
		n, err := humanize.ParseBytes(size)
		if err != nil {
			panic(err)
		}
		sizes = append(sizes, n)
	}

	return sizes
}

// PortNumber returns the port number assigned to the provided label, or falls
// back to the default value if none is assigned.
//
// TODO: we're getting this directly from an environment variable. We may want
//  to unpack in RunParams first.
func (rp *RunParams) PortNumber(label string, def string) string {
	v := strings.ToUpper(strings.TrimSpace(label)) + "_PORT"
	port, ok := os.LookupEnv(v)
	if !ok {
		return def
	}
	return port
}

// JSONParam unmarshals a JSON parameter in an arbitrary interface.
// It panics on error.
func (rp *RunParams) JSONParam(name string, v interface{}) {
	s, ok := rp.TestInstanceParams[name]
	if !ok {
		panic(fmt.Errorf("%s was not set", name))
	}

	if err := json.Unmarshal([]byte(s), v); err != nil {
		panic(err)
	}
}

// Copied from github.com/ipfs/testground/pkg/conv, because we don't want the
// SDK to depend on that package.
func ParseKeyValues(in []string) (res map[string]string, err error) {
	res = make(map[string]string, len(in))
	for _, d := range in {
		splt := strings.Split(d, "=")
		if len(splt) < 2 {
			return nil, fmt.Errorf("invalid key-value: %s", d)
		}
		res[splt[0]] = strings.Join(splt[1:], "=")
	}
	return res, nil
}

func unpackParams(packed string) map[string]string {
	spltparams := strings.Split(packed, "|")
	params := make(map[string]string, len(spltparams))
	for _, s := range spltparams {
		v := strings.Split(s, "=")
		if len(v) != 2 {
			continue
		}
		params[v[0]] = v[1]
	}
	return params
}

func toInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return v
}

func toBool(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}

// toNet might parse any input, so it is possible to get an error and nil return value
func toNet(s string) *ptypes.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil
	}
	return &ptypes.IPNet{IPNet: *ipnet}
}

// Try to parse the time.
// Failing to do so, return a zero value time
func toTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
