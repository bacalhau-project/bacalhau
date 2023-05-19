//go:build unit || !integration

package config

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *ConfigSuite) TestEnvWriter() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that the env writer works
	u, _ := uuid.NewRandom()
	summaryShellVariablesString := fmt.Sprintf("export TEST=%s", u.String())

	err := WriteRunInfoFile(ctx, summaryShellVariablesString)
	s.NoError(err)

	// Test the file contains the expected string
	contents, err := os.ReadFile(GetRunInfoFilePath())
	s.NoError(err)
	s.Equal(summaryShellVariablesString, string(contents))
}
