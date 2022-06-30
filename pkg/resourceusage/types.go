package resourceusage

// a record for the "amount" of compute resources an entity has / can consume / is using

type ResourceUsageConfig struct {
	// https://github.com/BTBurke/k8sresource string
	CPU string `json:"cpu"`
	// github.com/c2h5oh/datasize string
	Memory string `json:"memory"`
}

// these are the numeric values in bytes for ResourceUsageConfig
type ResourceUsageData struct {
	// cpu units
	CPU float64 `json:"cpu"`
	// bytes
	Memory uint64 `json:"memory"`
}
