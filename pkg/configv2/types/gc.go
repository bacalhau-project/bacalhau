package types

type TimeGC struct {
	// Threshold specifies the duration that data must exist before it becomes eligible for garbage collection.
	Threshold Duration `yaml:"Threshold,omitempty"`
	// Interval specifies the frequency at which the garbage collection process runs.
	Interval Duration `yaml:"Interval,omitempty"`
}
