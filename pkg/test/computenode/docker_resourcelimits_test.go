//go:build (integration || !unit) && !windows

package computenode

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/require"
)

type ComputeNodeDockerResourceLimitsSuite struct {
	scenario.ScenarioRunner
}

const CPU_LIMIT = "100m"
const MEMORY_LIMIT = "100mb"

func (suite *ComputeNodeDockerResourceLimitsSuite) TestDockerResourceLimitsCPU() {

	// this will give us a numerator and denominator that should end up at the
	// same 0.1 value that 100m means
	// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/managing_monitoring_and_updating_the_kernel/using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel#proc_controlling-distribution-of-cpu-time-for-applications-by-adjusting-cpu-bandwidth_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications
	testScenario := scenario.Scenario{
		Spec: model.Spec{
			Engine:   model.EngineDocker,
			Verifier: model.VerifierNoop,
			Resources: model.ResourceUsageConfig{
				CPU:    CPU_LIMIT,
				Memory: MEMORY_LIMIT,
			},
			Docker: model.JobSpecDocker{
				Image: "ubuntu",
				Entrypoint: []string{
					"bash",
					"-c",
					"cat /sys/fs/cgroup/cpu.max",
				},
			},
		},
		JobCheckers: scenario.WaitUntilComplete(1),
	}

	resultsDir := suite.RunScenario(testScenario)
	contents, err := os.ReadFile(filepath.Join(resultsDir, ipfs.DownloadFilenameStdout))
	require.NoError(suite.T(), err)

	result := string(contents)
	values := strings.Fields(result)

	numerator, err := strconv.Atoi(values[0])
	require.NoError(suite.T(), err)

	denominator, err := strconv.Atoi(values[1])
	require.NoError(suite.T(), err)

	var containerCPU float64 = 0
	if denominator > 0 {
		containerCPU = float64(numerator) / float64(denominator)
	}

	require.Equal(suite.T(), capacitymanager.ConvertCPUString(CPU_LIMIT), containerCPU, "the container reported CPU does not equal the configured limit")
}

func (suite *ComputeNodeDockerResourceLimitsSuite) TestDockerResourceLimitsMemory() {
	testScenario := scenario.Scenario{
		Spec: model.Spec{
			Engine:   model.EngineDocker,
			Verifier: model.VerifierNoop,
			Resources: model.ResourceUsageConfig{
				CPU:    CPU_LIMIT,
				Memory: MEMORY_LIMIT,
			},
			Docker: model.JobSpecDocker{
				Image: "ubuntu",
				Entrypoint: []string{
					"bash",
					"-c",
					"cat /sys/fs/cgroup/memory.max",
				},
			},
		},
		JobCheckers: scenario.WaitUntilComplete(1),
	}

	resultsDir := suite.RunScenario(testScenario)
	contents, err := os.ReadFile(filepath.Join(resultsDir, ipfs.DownloadFilenameStdout))
	require.NoError(suite.T(), err)

	result := string(contents)
	intVar, err := strconv.Atoi(strings.TrimSpace(result))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), capacitymanager.ConvertMemoryString(MEMORY_LIMIT), uint64(intVar), "the container reported memory does not equal the configured limit")
}
