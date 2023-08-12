package config_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

func TestConfigLoad(t *testing.T) {
	err := config.SetViperDefaults(config.Production())
	require.NoError(t, err)

	err = config.InitConfig(t.TempDir())
	require.NoError(t, err)
	cfg, err := config.Get()
	require.NoError(t, err)
	out, err := json.MarshalIndent(cfg, "", "\t")
	require.NoError(t, err)
	t.Logf("%v", string(out))
}
