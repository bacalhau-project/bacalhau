package types

// Resource represents allocated computing resources.
// The resource values are specified in Kubernetes format.
type Resource struct {
	// CPU specifies the amount of CPU allocated, in Kubernetes format (e.g., "100m" for 100 millicores).
	CPU string
	// Memory specifies the amount of memory allocated, in Kubernetes format (e.g., "1Gi" for 1 Gibibyte).
	Memory string
	// Disk specifies the amount of disk space allocated, in Kubernetes format (e.g., "10Gi" for 10 Gibibytes).
	Disk string
	// GPU specifies the amount of GPU resources allocated, in Kubernetes format (e.g., "1" for 1 GPU).
	GPU string
}
