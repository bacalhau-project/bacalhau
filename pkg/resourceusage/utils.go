package resourceusage

import (
	"runtime"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/c2h5oh/datasize"
	"github.com/pbnjay/memory"
)

// allow Mi, Gi to mean Mb, Gb
// remove spaces
// lowercase
func convertBytesString(st string) string {
	st = strings.ToLower(st)
	st = strings.ReplaceAll(st, "i", "b")
	st = strings.ReplaceAll(st, " ", "")
	return st
}

func ParseResourceUsageConfig(usage ResourceUsageConfig) (ResourceUsageData, error) {
	data := ResourceUsageData{}

	cpu, err := k8sresource.NewCPUFromString(convertBytesString(usage.CPU))
	if err != nil {
		return data, err
	}

	memory, err := datasize.ParseString(convertBytesString(usage.Memory))
	if err != nil {
		return data, err
	}

	data.CPU = cpu.ToFloat64()
	data.Memory = memory.Bytes()

	return data, nil
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
