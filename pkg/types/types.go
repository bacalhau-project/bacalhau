package types

// TODO: migrate all of these API types to publicapi

type ResultsList struct {
	Node   string `json:"node"`
	Cid    string `json:"cid"`
	Folder string `json:"folder"`
}

// Struct to report from the healthz endpoint
type HealthInfo struct {
	DiskFreeSpace FreeSpace `json:"FreeSpace"`
}

type FreeSpace struct {
	TMP  MountStatus `json:"tmp"`
	ROOT MountStatus `json:"root"`
}

// Creating structure for DiskStatus
type MountStatus struct {
	All  uint64 `json:"All"`
	Used uint64 `json:"Used"`
	Free uint64 `json:"Free"`
}

// Struct to report for VarZ
type VarZ struct {
	// TODO: #241 Fill in with varz to report
}
