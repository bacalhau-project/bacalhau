package types

type FeatureFlags struct {
	// ExecTranslation enables the execution translation feature.
	ExecTranslation bool `yaml:"ExecTranslation,omitempty" json:"ExecTranslation,omitempty"`
}
