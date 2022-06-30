package resourceusage

import (
	"runtime"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/c2h5oh/datasize"
	"github.com/pbnjay/memory"
)

func NewDefaultResourceUsageConfig() ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    "",
		Memory: "",
	}
}

func NewResourceUsageConfig(cpu, memory string) ResourceUsageConfig {
	return ResourceUsageConfig{
		CPU:    cpu,
		Memory: memory,
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

func ConvertCpuStringWithError(val string) (float64, error) {
	if val == "" {
		return 0, nil
	}
	cpu, err := k8sresource.NewCPUFromString(convertBytesString(val))
	if err != nil {
		return 0, err
	}
	return cpu.ToFloat64(), nil
}

func ConvertCpuString(val string) float64 {
	ret, err := ConvertCpuStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ConvertMemoryStringWithError(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}
	memory, err := datasize.ParseString(convertBytesString(val))
	if err != nil {
		return 0, err
	}
	return memory.Bytes(), nil
}

func ConvertMemoryString(val string) uint64 {
	ret, err := ConvertMemoryStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ParseResourceUsageConfig(usage ResourceUsageConfig) (ResourceUsageData, error) {
	return ResourceUsageData{
		CPU:    ConvertCpuString(usage.CPU),
		Memory: ConvertMemoryString(usage.Memory),
	}, nil
}

func GetResourceUsageConfig(usage ResourceUsageData) (ResourceUsageConfig, error) {
	config := ResourceUsageConfig{}

	cpu := k8sresource.NewCPUFromFloat(usage.CPU)

	config.CPU = cpu.ToString()
	config.Memory = (datasize.ByteSize(usage.Memory) * datasize.B).String()

	return config, nil
}

// what resources does this compute node actually have?
func GetSystemResources() (ResourceUsageData, error) {
	return ResourceUsageData{
		CPU:    float64(runtime.NumCPU()),
		Memory: memory.FreeMemory(),
	}, nil
}

// given a "required" usage and a "limit" of usage - can we run the requirement
func CompareUsageConfigs(wantsConfig, limitConfig ResourceUsageConfig) (bool, error) {

	// if there are no limits then everything goes
	if limitConfig.CPU == "" && limitConfig.Memory == "" {
		return true, nil
	}

	// if there are some limits and there are zero values for "wants"
	// we deny the job because we can't know if it would exceed our limit
	if wantsConfig.CPU == "" && wantsConfig.Memory == "" && (limitConfig.CPU != "" || limitConfig.Memory != "") {
		return false, nil
	}

	wants, err := ParseResourceUsageConfig(wantsConfig)
	if err != nil {
		return false, err
	}
	limit, err := ParseResourceUsageConfig(limitConfig)
	if err != nil {
		return false, err
	}

	return wants.CPU <= limit.CPU && wants.Memory <= limit.Memory, nil
}
