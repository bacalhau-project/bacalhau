package model

type EngineSpec struct {
	// TODO remove enum type in favor of string. Context here: https://www.notion.so/pl-strflt/Job-Schema-521ba6cdc06b4bdb940dbb151c576882?pvs=4#59b5896e974c45bb927a49f3dcb77306
	Type   Engine                 `json:"Type,omitempty"`
	Params map[string]interface{} `json:"Params,omitempty"`
}

func (e EngineSpec) AsWasmSpec() (*JobSpecWasm, error) {
	panic("TODO")
}

func (e EngineSpec) AsLanguageSpec() (*JobSpecLanguage, error) {
	panic("TODO")
}
