//go:build integration || !unit

package local_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/stretchr/testify/suite"
)

const defaultHost = "127.0.0.1"
const defaultPort = 6001

type PublisherTestSuite struct {
	ctx     context.Context
	baseDir string
	pub     *local.Publisher
	suite.Suite
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, &PublisherTestSuite{})
}

func (s *PublisherTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.baseDir = s.T().TempDir()
	s.pub = local.NewLocalPublisher(s.ctx, s.baseDir, defaultHost, defaultPort)
}

func (s *PublisherTestSuite) TestCopyFolder() {
	source := s.T().TempDir()

	err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("test"), 0644)
	s.Require().NoError(err)

	err = os.Mkdir(filepath.Join(source, "subdir"), 0755)
	s.Require().NoError(err)

	err = os.WriteFile(filepath.Join(source, "subdir", "file.txt"), []byte("test"), 0644)
	s.Require().NoError(err)

	exec := models.Execution{
		ID:    "eid",
		JobID: "jid",
	}

	cfg, err := s.pub.PublishResult(s.ctx, &exec, source)
	s.Require().NoError(err)
	s.NotNil(cfg)

	expected := fmt.Sprintf("http://%s:%d/jid/eid", defaultHost, defaultPort)
	s.Require().Equal(expected, cfg.Params["URL"])

	s.Require().FileExists(filepath.Join(s.baseDir, "jid", "eid", "file.txt"))
	s.Require().FileExists(filepath.Join(s.baseDir, "jid", "eid", "subdir", "file.txt"))
}
