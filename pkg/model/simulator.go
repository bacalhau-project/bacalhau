package model

type SimulatorConfigCompute struct {
	IsBadActor bool
}

type SimulatorConfigRequester struct {
	IsBadActor bool
}

type SimulatorConfig struct {
	Compute   SimulatorConfigCompute
	Requester SimulatorConfigRequester
}
