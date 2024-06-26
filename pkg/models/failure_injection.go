package models

type FailureInjectionRequesterConfig struct {
	IsBadActor bool `yaml:"IsBadActor"`
}
type FailureInjectionComputeConfig struct {
	IsBadActor bool
}
