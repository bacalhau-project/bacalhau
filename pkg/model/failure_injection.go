package model

type FailureInjectionComputeConfig struct {
	IsBadActor bool
}

type FailureInjectionRequesterConfig struct {
	IsBadActor bool
}

type FailureInjectionConfig struct {
	Compute   FailureInjectionComputeConfig
	Requester FailureInjectionRequesterConfig
}
