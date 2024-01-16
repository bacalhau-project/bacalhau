//go:build unit || !integration

package webui_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	types2 "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	utilserve "github.com/bacalhau-project/bacalhau/pkg/util/serve"
)

type tWebUISuite struct {
	*utilserve.ServeSuite
}

func TestWebUISuite(t *testing.T) {
	suite.Run(t, &tWebUISuite{ServeSuite: new(utilserve.ServeSuite)})
}

func (s *tWebUISuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	fsRepo := setup.SetupBacalhauRepoForTesting(s.T())
	repoPath, err := fsRepo.Path()
	s.Require().NoError(err)
	s.RepoPath = repoPath

	var cancel context.CancelFunc
	s.Ctx, cancel = context.WithTimeout(context.Background(), utilserve.MaxTestTime)
	s.T().Cleanup(func() {
		cancel()
	})

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(s.Ctx)
	})

	node, err := ipfs.NewNodeWithConfig(s.Ctx, cm, types2.IpfsConfig{PrivateInternal: true})
	s.Require().NoError(err)
	s.IPFSPort = node.APIPort
}
func (s *tWebUISuite) serveForWebUI(extraArgs ...string) (uint16, error) {
	webUIPort, err := freeport.GetFreePort()
	if err != nil {
		s.T().Fatal(err, "Could not get port for web-ui")
	}

	extraArgs = append(extraArgs, "--web-ui", "--web-ui-port", fmt.Sprintf("%d", webUIPort))

	// If the slice contains RETURN_ERROR_FLAG, take it out of the array and set returnError to true
	// peer set to "none" to avoid accidentally talking to production endpoints (even though it's default)
	// private-internal-ipfs to avoid accidentally talking to public IPFS nodes (even though it's default)
	returnError, _, err := utilserve.StartServerForTesting(s.ServeSuite, extraArgs)

	if returnError {
		return uint16(webUIPort), err
	}
	s.NoError(err)
	return uint16(webUIPort), nil
}

// Begin WebUI Tests
func (s *tWebUISuite) Test200ForNotStartingWebUI() {
	webUIPort, err := s.serveForWebUI()
	s.Require().NoError(err, "Error starting server")

	content, statusCode, err := utilserve.CurlEndpoint(s.Ctx, fmt.Sprintf("http://127.0.0.1:%d/", webUIPort))
	_ = content
	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *tWebUISuite) Test200ForRoot() {
	webUIPort, err := s.serveForWebUI()
	s.Require().NoError(err, "Error starting server")

	_, statusCode, err := utilserve.CurlEndpoint(s.Ctx, fmt.Sprintf("http://127.0.0.1:%d/", webUIPort))
	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *tWebUISuite) Test200ForSwagger() {
	swaggerContentString := "SwaggerUIBundle"

	// Create table for testing against swagger with Path, StatusCode, content to grep for
	swaggerTests := []struct {
		Path       string
		StatusCode int
		Content    string
	}{
		{"/swagger/", http.StatusOK, swaggerContentString},
		{"/swagger/index.html", http.StatusOK, swaggerContentString},
		{"/swagger/BAD_PATH", http.StatusNotFound, ""},
		{"/swagger/swagger.json", http.StatusOK, "\"swagger\": \"2.0\","},
	}

	webUIPort, err := s.serveForWebUI()
	s.Require().NoError(err, "Error starting server")

	for _, tt := range swaggerTests {
		content, statusCode, err := utilserve.CurlEndpoint(s.Ctx, fmt.Sprintf("http://127.0.0.1:%d%s", webUIPort, tt.Path))
		s.Require().NoError(err, "Error curling swagger endpoint")
		s.Require().Equal(tt.StatusCode, statusCode, fmt.Sprintf("%s: Did not return correct status code.", tt.Path))
		if tt.Content != "" {
			s.T().Logf("Content: %s", string(content))
			s.Require().Contains(string(content), tt.Content, fmt.Sprintf("%s: Did not return correct content.", tt.Path))
		}
	}
}
