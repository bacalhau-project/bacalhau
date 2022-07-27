package capacitymanager

// a record for the "amount" of compute resources an entity has / can consume / is using

type ResourceUsageConfig struct {
	// https://github.com/BTBurke/k8sresource string
	CPU string `json:"cpu" yaml:"cpu"`
	// github.com/c2h5oh/datasize string
	Memory string `json:"memory" yaml:"memory"`
	// github.com/c2h5oh/datasize string
	Disk string `json:"disk" yaml:"disk"`
	GPU  string `json:"gpu" yaml:"gpu"` // unsigned integer string
}

// these are the numeric values in bytes for ResourceUsageConfig
type ResourceUsageData struct {
	// cpu units
	CPU float64 `json:"cpu"`
	// bytes
	Memory uint64 `json:"memory"`
	// bytes
	Disk uint64 `json:"disk"`
	GPU  uint64 `json:"gpu"` // Support whole GPUs only, like https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/
}

type ResourceUsageProfile struct {
	// how many resources does the job want to consume
	Job ResourceUsageData `json:"job"`
	// how many resources is the system currently using
	SystemUsing ResourceUsageData `json:"system_using"`
	// what is the total amount of resources available to the system
	SystemTotal ResourceUsageData `json:"system_total"`
}
