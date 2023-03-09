//go:build unit || !integration

package logger

import (
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/logger/testpackage/subpackage/subsubpackage"
	ipfslog2 "github.com/ipfs/go-log/v2"
	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	kuboRepo "github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLogging(t *testing.T) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger

	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
	})

	var logging strings.Builder
	configureLogging(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.Out = &logging
		w.NoColor = true
	}))

	subsubpackage.TestLog("testing error logging", "testing message")

	actual := logging.String()
	// Like 12:47:40.875 | ERR pkg/logger/testpackage/subpackage/subsubpackage/testutil.go:12 > testing message error="testing error logging" [stack:[{"func":"TestLog","line":"10","source":"testutil.go"},{"func":"TestConfigureLogging","line":"27","source":"logger_test.go"},...]]
	t.Log(actual)

	assert.Contains(t, actual, "testing message", "Log statement doesn't contain the log message")
	assert.Contains(t, actual, `error="testing error logging"`, "Log statement doesn't contain the logged error")
	assert.Contains(t, actual, "pkg/logger/testpackage/subpackage/subsubpackage/testutil.go", "Log statement doesn't contain the full package path")
	assert.Contains(t, actual, `stack:[{"func":"TestLog","line":`, "Log statement didn't automatically include the error's stacktrace")
}

// TestConfigureIpfsLogging checks that we configure IPFS logging correctly, forwarding logging to zerolog.
func TestConfigureIpfsLogging(t *testing.T) {
	ipfslog2.SetupLogging(ipfslog2.Config{
		// This would normally be done by setting the "GOLOG_LOG_LEVEL" environment variable to "DEBUG"
		Level: ipfslog2.LevelDebug,
	})

	oldLevel := os.Getenv("LOG_LEVEL")
	t.Cleanup(func() {
		assert.NoError(t, os.Setenv("LOG_LEVEL", oldLevel))
	})
	require.NoError(t, os.Setenv("LOG_LEVEL", "DEBUG"))

	var logging strings.Builder
	configureLogging(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.Out = &logging
		w.NoColor = true
	}))

	triggerIPFSLogging(t)

	l := ipfslog2.Logger("name")
	l.With("hello", "world", "err", errors.New("example")).Error("test")

	actual := logging.String()
	// Like 12:06:50.55 | ERR pkg/logger/logger_test.go:52 > test [err:example] [hello:world] [logger-name:name]
	t.Log(actual)
	t.Log(os.Getenv("GOPATH"))

	assert.Regexp(t, regexp.MustCompile(`ERR pkg/logger/logger_test.go:\d* > test`), actual)
	assert.Contains(t, actual, "[hello:world]")
	assert.Contains(t, actual, "[logger-name:name]")
	assert.Contains(t, actual, "[err:example]")

	assert.Regexp(t,
		regexp.MustCompile(`DBG github.com/libp2p/go-libp2p@v[\d.]*/config/log.go`),
		actual,
		"Logging from IPFS or libp2p should look like a path within the dependency",
	)
}

func triggerIPFSLogging(t *testing.T) {
	// Do something to get IPFS or libp2p to log something, such as spinning up IPFS in-process.
	plugins, err := loader.NewPluginLoader("")
	require.NoError(t, err)

	require.NoError(t, plugins.Initialize())
	require.NoError(t, plugins.Inject())

	t.Cleanup(func() {
		// Just want the logging
		_ = plugins.Close()
	})

	repoPath := t.TempDir()

	var repo kuboRepo.Repo
	cfg, err := config.Init(io.Discard, 2048)
	require.NoError(t, err)
	require.NoError(t, config.Profiles["test"].Transform(cfg))

	cfg.AutoNAT.ServiceMode = config.AutoNATServiceDisabled
	cfg.Swarm.EnableHolePunching = config.False
	cfg.Swarm.DisableNatPortMap = true
	cfg.Swarm.RelayClient.Enabled = config.False
	cfg.Swarm.RelayService.Enabled = config.False
	cfg.Swarm.Transports.Network.Relay = config.False
	cfg.Discovery.MDNS.Enabled = false
	cfg.Addresses.Gateway = []string{"/ip4/0.0.0.0/tcp/0"}
	cfg.Addresses.API = []string{"/ip4/0.0.0.0/tcp/0"}
	cfg.Addresses.Swarm = []string{"/ip4/0.0.0.0/tcp/0"}
	cfg.Peering = config.Peering{
		Peers: nil,
	}

	require.NoError(t, fsrepo.Init(repoPath, cfg))

	repo, err = fsrepo.Open(repoPath)
	require.NoError(t, err)

	nodeOptions := &core.BuildCfg{
		Repo:    repo,
		Online:  true,
		Routing: libp2p.DHTClientOption,
	}

	node, err := core.NewNode(context.Background(), nodeOptions)
	require.NoError(t, err)

	t.Cleanup(func() {
		// Just want the logging
		_ = node.Close()
	})
}
