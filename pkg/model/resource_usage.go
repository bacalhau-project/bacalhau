package model

// a record for the "amount" of compute resources an entity has / can consume / is using

type ResourceUsageConfig struct {
	// https://github.com/BTBurke/k8sresource string
	CPU string `json:"CPU,omitempty" yaml:"CPU,omitempty"`
	// github.com/c2h5oh/datasize string
	Memory string `json:"Memory,omitempty" yaml:"Memory,omitempty"`
	// github.com/c2h5oh/datasize string

	Disk string `json:"Disk,omitempty" yaml:"Disk,omitempty"`
	GPU  string `json:"GPU" yaml:"GPU"` // unsigned integer string

}

// these are the numeric values in bytes for ResourceUsageConfig
type ResourceUsageData struct {
	// cpu units
	CPU float64 `json:"CPU,omitempty"`
	// bytes
	Memory uint64 `json:"Memory,omitempty"`
	// bytes
	Disk uint64 `json:"Disk,omitempty"`
	GPU  uint64 `json:"GPU,omitempty"` // Support whole GPUs only, like https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/
}

type ResourceUsageProfile struct {
	// how many resources does the job want to consume
	Job ResourceUsageData `json:"Job,omitempty"`
	// how many resources is the system currently using
	SystemUsing ResourceUsageData `json:"SystemUsing,omitempty"`
	// what is the total amount of resources available to the system
	SystemTotal ResourceUsageData `json:"SystemTotal,omitempty"`
}
