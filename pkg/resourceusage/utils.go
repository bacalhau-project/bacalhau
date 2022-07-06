package resourceusage

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"

	"github.com/BTBurke/k8sresource"
	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/pbnjay/memory"
)

func NewDefaultResourceUsageConfig() ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    "",
		Memory: "",
		Disk:   "",
	}
}

func NewResourceUsageConfig(cpu, mem, disk string) ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    cpu,
		Memory: mem,
		Disk:   disk,
	}
}

// allow Mi, Gi to mean Mb, Gb
// remove spaces
// lowercase
func convertBytesString(st string) string {
	st = strings.ToLower(st)
	st = strings.ReplaceAll(st, "i", "b")
	st = strings.ReplaceAll(st, " ", "")
	return st
}

func ConvertCPUStringWithError(val string) (float64, error) {
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
	ret, err := ConvertCPUStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ConvertMemoryStringWithError(val string) (uint64, error) {
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
	ret, err := ConvertMemoryStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ParseResourceUsageConfig(usage ResourceUsageConfig) ResourceUsageData {
	return ResourceUsageData{
		CPU:    ConvertCPUString(usage.CPU),
		Memory: ConvertMemoryString(usage.Memory),
		Disk:   ConvertMemoryString(usage.Disk),
	}
}

func GetResourceUsageConfig(usage ResourceUsageData) (ResourceUsageConfig, error) {
	config := ResourceUsageConfig{}

	cpu := k8sresource.NewCPUFromFloat(usage.CPU)

	config.CPU = cpu.ToString()
	config.Memory = (datasize.ByteSize(usage.Memory) * datasize.B).String()
	config.Disk = (datasize.ByteSize(usage.Disk) * datasize.B).String()

	return config, nil
}

// get free disk space for storage path
// returns bytes
func getFreeDiskSpace(path string) (uint64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return 0, err
	}
	return fs.Bfree * uint64(fs.Bsize), nil
}

// what resources does this compute node actually have?
func GetSystemResources(limitConfig ResourceUsageConfig) (ResourceUsageData, error) {
	diskSpace, err := getFreeDiskSpace(config.GetStoragePath())
	if err != nil {
		return ResourceUsageData{}, err
	}

	// the actual resources we have
	data := ResourceUsageData{
		CPU:    float64(runtime.NumCPU()),
		Memory: memory.TotalMemory(),
		Disk:   diskSpace,
	}

	parsedLimitConfig := ParseResourceUsageConfig(limitConfig)

	if parsedLimitConfig.CPU > 0 {
		if parsedLimitConfig.CPU > data.CPU {
			return data, fmt.Errorf(
				"you cannot configure more CPU than you have on this node: configured %f, have %f",
				parsedLimitConfig.CPU, data.CPU,
			)
		}
		data.CPU = parsedLimitConfig.CPU
	}

	if parsedLimitConfig.Memory > 0 {
		if parsedLimitConfig.Memory > data.Memory {
			return data, fmt.Errorf(
				"you cannot configure more Memory than you have on this node: configured %d, have %d",
				parsedLimitConfig.Memory, data.Memory,
			)
		}
		data.Memory = parsedLimitConfig.Memory
	}

	if parsedLimitConfig.Disk > 0 {
		if parsedLimitConfig.Disk > data.Disk {
			return data, fmt.Errorf(
				"you cannot configure more disk than you have on this node: configured %d, have %d",
				parsedLimitConfig.Disk, data.Disk,
			)
		}
		data.Disk = parsedLimitConfig.Disk
	}

	return data, nil
}

// given a "required" usage and a "limit" of usage - can we run the requirement
func CheckResourceRequirements(wants, limits ResourceUsageData) bool {
	// if there are no limits then everything goes
	if limits.CPU <= 0 && limits.Memory <= 0 && limits.Disk <= 0 {
		return true
	}
	// if there are some limits and there are zero values for "wants"
	// we deny the job because we can't know if it would exceed our limit
	if wants.CPU <= 0 && wants.Memory <= 0 && wants.Disk <= 0 && (limits.CPU > 0 || limits.Memory > 0 || limits.Disk > 0) {
		return false
	}
	return wants.CPU <= limits.CPU && wants.Memory <= limits.Memory && wants.Disk <= limits.Disk
}
