package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

func GetConfigForKey(key string, cfg interface{}) error {
	return viper.UnmarshalKey(key, &cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
}
