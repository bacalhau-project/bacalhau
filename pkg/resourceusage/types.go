package resourceusage

// a record for the "amount" of compute resources an entity has / can consume / is using
// https://github.com/BTBurke/k8sresource strings
type ResourceUsageData struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Disk   string `json:"disk"`
}
