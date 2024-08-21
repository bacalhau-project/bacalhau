package types

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func DecodeProviderConfig[P ProviderType](cfg ConfigProvider) (P, error) {
	// Create a new instance of P
	var target P

	// Access the kind statically
	kind := target.Kind()

	// Check if the config for the specified kind exists
	if !cfg.HasConfig(kind) {
		return target, fmt.Errorf("no config found for publisher: %s", kind)
	}
	data := cfg.ConfigMap()[kind]

	// Use mapstructure to decode the config into the target instance
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "mapstructure",
		Result:  &target,
	})
	if err != nil {
		return target, err
	}

	err = decoder.Decode(data)
	return target, err
}
