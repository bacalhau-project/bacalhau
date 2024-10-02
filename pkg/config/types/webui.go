package types

type WebUI struct {
	// Enabled indicates whether the Web UI is enabled.
	Enabled bool `yaml:"Enabled,omitempty" json:"Enabled,omitempty"`
	// Listen specifies the address and port on which the Web UI listens.
	Listen string `yaml:"Listen,omitempty" json:"Listen,omitempty"`
	// Backend specifies the address and port of the backend API server.
	// If empty, the Web UI will use the same address and port as the API server.
	Backend string `yaml:"Backend,omitempty" json:"Backend,omitempty"`
}
