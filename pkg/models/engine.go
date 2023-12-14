package models

func IsDefaultEngineType(kind string) bool {
	return kind == EngineDocker || kind == EngineNoop || kind == EngineWasm
}
