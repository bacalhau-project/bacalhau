//go:build integration || !unit

package local_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
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
	var err error
	s.pub, err = local.NewLocalPublisher(s.ctx, s.baseDir, defaultHost, defaultPort)
	s.Require().NoError(err)
}

func (s *PublisherTestSuite) TestAddressResolving() {
	ctx := context.Background()

	s.Require().Equal("127.0.0.1", local.ResolveAddress(ctx, "127.0.0.1"), "address did not resolve to itself")
	s.Require().Equal("192.168.1.100", local.ResolveAddress(ctx, "192.168.1.100"), "address did not resolve to itself")
	s.Require().Equal("127.0.0.1", local.ResolveAddress(ctx, "local"))
}

func (s *PublisherTestSuite) TestPublishFolder() {
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

	expected := fmt.Sprintf("http://%s:%d/eid.tar.gz", defaultHost, defaultPort)
	s.Require().Equal(expected, cfg.Params["URL"])
}
