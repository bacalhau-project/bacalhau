package types

type UpdateConfig struct {
	// Interval specifies the time between update checks, when set to 0 update checks are not performed.
	Interval Duration `yaml:"Interval,omitempty" json:"Interval,omitempty"`
}
