package analytics

type Resource struct {
	CPUUnits    float64   `json:"cpu_units,omitempty"`
	MemoryBytes uint64    `json:"memory_bytes,omitempty"`
	DiskBytes   uint64    `json:"disk_bytes,omitempty"`
	GPUCount    uint64    `json:"gpu_count,omitempty"`
	GPUTypes    []GPUInfo `json:"gpu_types,omitempty"`
}

type GPUInfo struct {
	Name   string `json:"name,omitempty"`
	Vendor string `json:"vendor,omitempty"`
}
