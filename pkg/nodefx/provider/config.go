package provider

type Config struct {
	Storage   map[string][]byte
	Executor  map[string][]byte
	Publisher map[string][]byte
}
