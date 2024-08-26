//go:build unit || !integration

package serve_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	apitest "github.com/bacalhau-project/bacalhau/pkg/publicapi/test"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"

	cmd2 "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

const maxServeTime = 30 * time.Hour
const maxTestTime = 60 * time.Hour
const RETURN_ERROR_FLAG = "RETURN_ERROR"

type ServeSuite struct {
	suite.Suite

	out, err strings.Builder

	ctx      context.Context
	repoPath string
	config   types2.Bacalhau
}

func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}

func (s *ServeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	/*
		fsRepo, c := setup.SetupBacalhauRepoForTesting(s.T())
		repoPath, err := fsRepo.Path()
		s.Require().NoError(err)
		s.repoPath = repoPath
		s.config = c

	*/

	var cancel context.CancelFunc
	s.ctx, cancel = context.WithTimeout(context.Background(), maxTestTime)
	s.T().Cleanup(func() {
		cancel()
	})

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(s.ctx)
	})
}

func (s *ServeSuite) serve(extraArgs ...string) (uint16, error) {
	// If the slice contains RETURN_ERROR_FLAG, take it out of the array and set returnError to true
	returnError := false
	for i, arg := range extraArgs {
		if arg == RETURN_ERROR_FLAG {
			extraArgs = append(extraArgs[:i], extraArgs[i+1:]...)
			returnError = true
			break
		}
	}

	bigPort, err := network.GetFreePort()
	s.Require().NoError(err)
	port := uint16(bigPort)

	cmd := cmd2.NewRootCmd()
	cmd.SetOut(&s.out)
	cmd.SetErr(&s.err)

	args := []string{
		"serve",
		"--repo", s.T().TempDir(),
		"--api-address", fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	args = append(args, extraArgs...)
	cmd.SetArgs(args)
	s.T().Logf("Command to execute: %q", args)

	ctx, cancel := context.WithTimeout(s.ctx, maxServeTime)
	errs, ctx := errgroup.WithContext(ctx)
	s.T().Cleanup(func() {
		cancel()
	})

	errs.Go(func() error {
		_, err := cmd.ExecuteContextC(ctx)
		if returnError {
			return err
		}
		s.NoError(err)
		return nil
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- errs.Wait()
	}()

	t := time.NewTicker(50 * time.Millisecond)

	defer t.Stop()
	for {
		select {
		case e := <-errCh:
			s.FailNow("Server raised an error during startup: %s", e)
		case <-ctx.Done():
			if returnError {
				return 0, errs.Wait()
			}
			s.FailNow("Server did not start in time")

		case <-t.C:
			livezText, statusCode, _ := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/api/v1/livez", port))
			if string(livezText) == "OK" && statusCode == http.StatusOK {
				return port, nil
			}
		}
	}
}

func (s *ServeSuite) curlEndpoint(URL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(s.ctx, "GET", URL, nil)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	req.Header.Set("Accept", "application/json")
	client := http.DefaultClient
	if strings.HasPrefix(URL, "https") {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	defer closer.DrainAndCloseWithLogOnError(s.ctx, "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err

	}
	return responseText, resp.StatusCode, nil
}
func (s *ServeSuite) TestHealthcheck() {
	port, _ := s.serve()
	healthzText, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/api/v1/healthz", port))
	s.Require().NoError(err)

	var healthzJSON types.HealthInfo
	s.Require().NoError(marshaller.JSONUnmarshalWithMax(healthzText, &healthzJSON), "Error unmarshalling healthz JSON.")
	s.Require().Greater(int(healthzJSON.DiskFreeSpace.ROOT.All), 0, "Did not report DiskFreeSpace > 0.")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *ServeSuite) TestCanSubmitJob() {
	docker.MustHaveDocker(s.T())
	port, err := s.serve("--orchestrator", "--compute")
	s.Require().NoError(err)

	client := clientv2.New(fmt.Sprintf("http://127.0.0.1:%d", port))
	s.Require().NoError(apitest.WaitForAlive(s.ctx, client))

	job := &models.Job{
		Name:  s.T().Name(),
		Type:  models.JobTypeBatch,
		Count: 1,
		Tasks: []*models.Task{
			{
				Name: s.T().Name(),
				Engine: &models.SpecConfig{
					Type:   models.EngineNoop,
					Params: make(map[string]interface{}),
				},
			},
		},
	}
	job.Normalize()
	_, err = client.Jobs().Put(s.ctx, &apimodels.PutJobRequest{
		Job: job,
	})
	s.NoError(err)
}

// Begin WebUI Tests
func (s *ServeSuite) Test200ForNotStartingWebUI() {
	port, err := s.serve()
	s.Require().NoError(err, "Error starting server")

	content, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/", port))
	_ = content

	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}

func (s *ServeSuite) Test200ForRoot() {
	webUIPort, err := network.GetFreePort()
	if err != nil {
		s.T().Fatal(err, "Could not get port for web-ui")
	}
	_, err = s.serve("-c", "webui.enabled=true", "-c", fmt.Sprintf("webui.listen=127.0.0.1:%d", webUIPort))
	s.Require().NoError(err, "Error starting server")

	_, statusCode, err := s.curlEndpoint(fmt.Sprintf("http://127.0.0.1:%d/", webUIPort))
	s.Require().NoError(err, "Error curling root endpoint")
	s.Require().Equal(http.StatusOK, statusCode, "Did not return 200 OK.")
}
