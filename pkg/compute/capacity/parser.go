package capacity

import (
	"strconv"
	"strings"

	"github.com/BTBurke/k8sresource"
	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

func ParseResourceUsageConfig(usage model.ResourceUsageConfig) model.ResourceUsageData {
	return model.ResourceUsageData{
		CPU:    ConvertCPUString(usage.CPU),
		Memory: ConvertBytesString(usage.Memory),
		Disk:   ConvertBytesString(usage.Disk),
		GPU:    ConvertGPUString(usage.GPU),
	}
}
func ConvertCPUString(val string) float64 {
	ret, err := convertCPUStringWithError(val)
	if err != nil {
		return 0
	}
	return ret
}

func ConvertBytesString(val string) uint64 {
	ret, err := convertBytesStringWithError(val)
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
	mem, err := datasize.ParseString(sanitizeBytesString(val))
	if err != nil {
		return 0, err
	}
	return mem.Bytes(), nil
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
