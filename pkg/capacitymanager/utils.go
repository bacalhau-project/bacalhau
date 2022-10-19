package capacitymanager

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/pbnjay/memory"
	"github.com/ricochet2200/go-disk-usage/du"
)

// this is used mainly for tests to be deterministic
// or for tests to say "I know I don't have GPUs I am pretenting I do"
func SetIgnorePhysicalResources(value string) {
	os.Setenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT", value)
}

func shouldIgnorePhysicalResources() bool {
	return os.Getenv("BACALHAU_CAPACITY_MANAGER_OVER_COMMIT") != ""
}

// NvidiaCLI is the path to the Nvidia helper binary
const NvidiaCLI = "nvidia-container-cli"

// allow Mi, Gi to mean Mb, Gb
// remove spaces
// lowercase
func convertBytesString(st string) string {
	st = strings.ToLower(st)
	st = strings.ReplaceAll(st, "i", "b")
	st = strings.ReplaceAll(st, " ", "")
	return st
}

func convertCPUStringWithError(val string) (float64, error) {
	if val == "" {
		return 0, nil
	}
	cpu, err := k8sresource.NewCPUFromString(convertBytesString(val))
	if err != nil {
		return 0, err
	}
	return cpu.ToFloat64(), nil
}

func ConvertCPUString(val string) float64 {
	ret, err := convertCPUStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func convertMemoryStringWithError(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}
	mem, err := datasize.ParseString(convertBytesString(val))
	if err != nil {
		return 0, err
	}
	return mem.Bytes(), nil
}

func ConvertMemoryString(val string) uint64 {
	ret, err := convertMemoryStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ConvertGPUString(val string) uint64 {
	ret, err := strconv.ParseUint(val, 10, 64) //nolint:gomnd
	if err != nil {
		return 0
	}
	return ret
}

func ParseResourceUsageConfig(usage model.ResourceUsageConfig) model.ResourceUsageData {
	return model.ResourceUsageData{
		CPU:    ConvertCPUString(usage.CPU),
		Memory: ConvertMemoryString(usage.Memory),
		Disk:   ConvertMemoryString(usage.Disk),
		GPU:    ConvertGPUString(usage.GPU),
	}
}

// get free disk space for storage path
// returns bytes
func getFreeDiskSpace(path string) (uint64, error) {
	usage := du.NewDiskUsage(path)
	if usage == nil {
		return 0, fmt.Errorf("getFreeDiskSpace: unable to get disk space for path %s", path)
	}
	return usage.Free(), nil
}

// numSystemGPUs wraps nvidia-container-cli to get the number of GPUs
func numSystemGPUs() (uint64, error) {
	nvidiaPath, err := exec.LookPath(NvidiaCLI)
	if err != nil {
		// If the NVIDIA CLI is not installed, we can't know the number of GPUs, assume zero
		if (err.(*exec.Error)).Unwrap() == exec.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	args := []string{
		"info",
		"--csv",
	}
	cmd := exec.Command(nvidiaPath, args...)
	resp, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Parse output of nvidia-container-cli command
	lines := strings.Split(string(resp), "\n")
	deviceInfoFlag := false
	numDevices := uint64(0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "Device Index") {
			deviceInfoFlag = true
			continue
		}
		if deviceInfoFlag {
			numDevices += 1
		}
	}

	fmt.Println(numDevices)
	return numDevices, nil
}

// what resources does this compute node actually have?
func getSystemResources(limitConfig model.ResourceUsageConfig) (model.ResourceUsageData, error) {
	diskSpace, err := getFreeDiskSpace(config.GetStoragePath())
	if err != nil {
		return model.ResourceUsageData{}, err
	}
	gpus, err := numSystemGPUs()
	if err != nil {
		return model.ResourceUsageData{}, err
	}

	// the actual resources we have
	physcialResources := model.ResourceUsageData{
		CPU:    float64(runtime.NumCPU()),
		Memory: memory.TotalMemory(),
		Disk:   diskSpace,
		GPU:    gpus,
	}

	parsedLimitConfig := ParseResourceUsageConfig(limitConfig)

	if parsedLimitConfig.CPU > 0 {
		if parsedLimitConfig.CPU > physcialResources.CPU && !shouldIgnorePhysicalResources() {
			return physcialResources, fmt.Errorf(
				"you cannot configure more CPU than you have on this node: configured %f, have %f",
				parsedLimitConfig.CPU, physcialResources.CPU,
			)
		}
		physcialResources.CPU = parsedLimitConfig.CPU
	}

	if parsedLimitConfig.Memory > 0 {
		if parsedLimitConfig.Memory > physcialResources.Memory && !shouldIgnorePhysicalResources() {
			return physcialResources, fmt.Errorf(
				"you cannot configure more Memory than you have on this node: configured %d, have %d",
				parsedLimitConfig.Memory, physcialResources.Memory,
			)
		}
		physcialResources.Memory = parsedLimitConfig.Memory
	}

	if parsedLimitConfig.Disk > 0 {
		if parsedLimitConfig.Disk > physcialResources.Disk && !shouldIgnorePhysicalResources() {
			return physcialResources, fmt.Errorf(
				"you cannot configure more disk than you have on this node: configured %d, have %d",
				parsedLimitConfig.Disk, physcialResources.Disk,
			)
		}
		physcialResources.Disk = parsedLimitConfig.Disk
	}

	if parsedLimitConfig.GPU > 0 {
		if parsedLimitConfig.GPU > physcialResources.GPU && !shouldIgnorePhysicalResources() {
			return physcialResources, fmt.Errorf(
				"you cannot configure more GPU than you have on this node: configured %d, have %d",
				parsedLimitConfig.GPU, physcialResources.GPU,
			)
		}
		physcialResources.GPU = parsedLimitConfig.GPU
	}

	return physcialResources, nil
}

// given a "required" usage and a "limit" of usage - can we run the requirement
func checkResourceUsage(wants, limits model.ResourceUsageData) bool {
	zeroWants := wants.CPU <= 0 &&
		wants.Memory <= 0 &&
		wants.Disk <= 0 &&
		wants.GPU <= 0

	limitOverZero := limits.CPU > 0 ||
		limits.Memory > 0 ||
		limits.Disk > 0 ||
		limits.GPU > 0

	// if there are some limits and there are zero values for "wants"
	// we deny the job because we can't know if it would exceed our limit
	if zeroWants && limitOverZero {
		return false
	}

	return wants.CPU <= limits.CPU &&
		wants.Memory <= limits.Memory &&
		wants.Disk <= limits.Disk &&
		wants.GPU <= limits.GPU
}

func subtractResourceUsage(current, totals model.ResourceUsageData) model.ResourceUsageData {
	return model.ResourceUsageData{
		CPU:    totals.CPU - current.CPU,
		Memory: totals.Memory - current.Memory,
		Disk:   totals.Disk - current.Disk,
		GPU:    totals.GPU - current.GPU,
	}
}

// add the shards in random order so we get some kind of general coverage across
// the network - otherwise all nodes are racing each other for the same shards
func GenerateShardIndexes(shardCount int, requirements model.ResourceUsageData) []int {
	shardIndexes := []int{}
	for i := 0; i < shardCount; i++ {
		shardIndexes = append(shardIndexes, i)
	}
	for i := range shardIndexes {
		j := rand.Intn(i + 1) //nolint:gosec
		shardIndexes[i], shardIndexes[j] = shardIndexes[j], shardIndexes[i]
	}
	return shardIndexes
}
