//go:build unit || !integration

package util_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
)

func TestSetupConfig(t *testing.T) {
	t.Setenv("BACALHAU_ENVIRONMENT", "production")

	testCases := []struct {
		name               string
		repoConfig         string
		xdgConfig          string
		flagConfig         string
		expectedClientHost string
	}{
		{
			name:               "No Parameters",
			expectedClientHost: configenv.Production.Node.ClientAPI.Host,
		},
		{
			name: "Config in Repo",
			repoConfig: `
Node:
  ClientAPI:
    Host: 1.1.1.1
`,
			expectedClientHost: "1.1.1.1",
		},
		{
			name: "Config in $XDG_CONFIG_HOME/bacalhau/config.yaml",
			xdgConfig: `
Node:
  ClientAPI:
    Host: 2.2.2.2
`,
			expectedClientHost: "2.2.2.2",
		},
		{
			name: "config in repo and $XDG_CONFIG_HOME/bacalhau/config.yaml",
			repoConfig: `
Node:
  ClientAPI:
    Host: 1.1.1.1
`,
			xdgConfig: `
Node:
  ClientAPI:
    Host: 2.2.2.2
`,
			expectedClientHost: "2.2.2.2",
		},
		{
			name: "config in repo and $XDG_CONFIG_HOME/bacalhau/config.yaml and flag",
			repoConfig: `
Node:
  ClientAPI:
    Host: 1.1.1.1
`,
			xdgConfig: `
Node:
  ClientAPI:
    Host: 2.2.2.2
`,
			flagConfig: `
Node:
  ClientAPI:
    Host: 3.3.3.3
`,
			expectedClientHost: "3.3.3.3",
		},
		{
			name: "config only from flag",
			flagConfig: `
Node:
  ClientAPI:
    Host: 3.3.3.3
`,
			expectedClientHost: "3.3.3.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset viper for each test run
			viper.Reset()
			defer viper.Reset()

			// ensure the repo path is always present
			v := viper.GetViper()
			repoPath := t.TempDir()
			v.Set("repo", repoPath)

			// create config files for test case if provided
			if tc.repoConfig != "" {
				_, err := createTempConfig(repoPath, tc.repoConfig, false)
				require.NoError(t, err)
			}

			if tc.xdgConfig != "" {
				xdgDir := t.TempDir()
				setXDGConfigEnv(t, xdgDir)
				_, err := createTempConfig(xdgDir, tc.xdgConfig, true)
				require.NoError(t, err)
			}

			cmd, err := setupTestCommand()
			require.NoError(t, err)

			if tc.flagConfig != "" {
				flagConfigDir := t.TempDir()
				flagConfigFile, err := createTempConfig(flagConfigDir, tc.flagConfig, false)
				require.NoError(t, err)
				cmd.SetArgs([]string{"--config", flagConfigFile})
			}

			// invoke the command to trigger flag parsing
			require.NoError(t, cmd.Execute())

			cfg, err := util.SetupConfig(cmd)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			bacalhauCfg, err := cfg.Current()
			require.NoError(t, err)

			assert.Equal(t, tc.expectedClientHost, bacalhauCfg.Node.ClientAPI.Host)
		})
	}
}

func setupTestCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	cmd.PersistentFlags().VarP(cliflags.NewConfigFlag(), "config", "c", "config file(s) or dot separated path(s) to config values")
	return cmd, nil
}

func createTempConfig(path, content string, xdg bool) (string, error) {
	if xdg {
		// NB: see impl of the function os.UserConfigDir() to understand why this is needed.
		switch runtime.GOOS {
		case "darwin", "ios":
			path = filepath.Join(path, "Library", "Application Support")
			if err := os.MkdirAll(path, 0777); err != nil {
				return "", err
			}
		case "plan9":
			path = filepath.Join(path, "lib")
			if err := os.MkdirAll(path, 0777); err != nil {
				return "", err
			}
		}
	}
	tmpfile, err := os.Create(filepath.Join(path, config.FileName))
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

func setXDGConfigEnv(t *testing.T, xdgDir string) {
	// NB: see impl of the function os.UserConfigDir() to understand why this is needed.
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", xdgDir)
	case "darwin", "ios":
		t.Setenv("HOME", xdgDir)
	case "plan9":
		t.Setenv("home", xdgDir)
	default: // Unix
		t.Setenv("XDG_CONFIG_HOME", xdgDir)
	}
}
