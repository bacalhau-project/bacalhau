package model

import (
	"strconv"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/dustin/go-humanize"
)

func ParseResourceUsageConfig(usage ResourceUsageConfig) (ResourceUsageData, error) {
	cpu, err := ConvertCPUString(usage.CPU)
	if err != nil {
		return ResourceUsageData{}, err
	}
	mem, err := ConvertBytesString(usage.Memory)
	if err != nil {
		return ResourceUsageData{}, err
	}
	disk, err := ConvertBytesString(usage.Disk)
	if err != nil {
		return ResourceUsageData{}, err
	}
	gpu, err := ConvertGPUString(usage.GPU)
	if err != nil {
		return ResourceUsageData{}, err
	}

	return ResourceUsageData{
		CPU:    cpu,
		Memory: mem,
		Disk:   disk,
		GPU:    gpu,
	}, nil
}
func ConvertCPUString(val string) (float64, error) {
	ret, err := convertCPUStringWithError(val)
	if err != nil {
		return 0, err
	}
	return ret, nil
}

func ConvertBytesString(val string) (uint64, error) {
	ret, err := convertBytesStringWithError(val)
	if err != nil {
		return 0, err
	}
	return ret, nil
}

func ConvertGPUString(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}
	ret, err := strconv.ParseUint(val, 10, 64) //nolint:gomnd
	if err != nil {
		return 0, err
	}
	return ret, nil
}

func convertCPUStringWithError(val string) (float64, error) {
	if val == "" {
		return 0, nil
	}
	cpu, err := k8sresource.NewCPUFromString(sanitizeBytesString(val))
	if err != nil {
		return 0, err
	}
	return cpu.ToFloat64(), nil
}

func convertBytesStringWithError(val string) (uint64, error) {
	if val == "" {
		return 0, nil
	}
	return humanize.ParseBytes(val)
}

// allow Mi, Gi to mean Mb, Gb
// remove spaces
// lowercase
func sanitizeBytesString(st string) string {
	st = strings.ToLower(st)
	st = strings.ReplaceAll(st, "i", "b")
	st = strings.ReplaceAll(st, " ", "")
	return st
}
