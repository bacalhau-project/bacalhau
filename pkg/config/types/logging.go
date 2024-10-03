package types

type Logging struct {
	// Level sets the logging level. One of: trace, debug, info, warn, error, fatal, panic.
	Level string `yaml:"Level,omitempty" json:"Level,omitempty"`
	// Mode specifies the logging mode. One of: default, json.
	Mode string `yaml:"Mode,omitempty" json:"Mode,omitempty"`
	// LogDebugInfoInterval specifies the interval for logging debug information.
	LogDebugInfoInterval Duration `yaml:"LogDebugInfoInterval,omitempty" json:"LogDebugInfoInterval,omitempty"`
}
