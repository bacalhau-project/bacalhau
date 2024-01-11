package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	cmd2 "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

const MaxServeTime = 15 * time.Second
const MaxTestTime = 1000 * time.Second
const RETURN_ERROR_FLAG = "RETURN_ERROR" //nolint:stylecheck
const TickerIncrements = 10 * time.Millisecond

type ServeSuite struct {
	suite.Suite

	Out, Err strings.Builder

	Ctx context.Context

	IPFSPort int
	RepoPath string
}

func StartServerForTesting(
	s *ServeSuite,
	extraArgs []string) (bool, uint16, error) {
	returnError := false
	for i, arg := range extraArgs {
		if arg == RETURN_ERROR_FLAG {
			extraArgs = append(extraArgs[:i], extraArgs[i+1:]...)
			returnError = true
			break
		}
	}

	bigPort, err := freeport.GetFreePort()
	if err != nil {
		return false, 0, err
	}
	port := uint16(bigPort)

	cmd := cmd2.NewRootCmd()
	cmd.SetOut(&s.Out)
	cmd.SetErr(&s.Err)

	args := []string{
		"serve",
		"--port", fmt.Sprint(port),
	}
	args = append(args, extraArgs...)

	cmd.SetArgs(args)
	s.T().Logf("Command to execute: %q", args)

	ctx, cancel := context.WithTimeout(s.Ctx, MaxServeTime)
	errs, ctx := errgroup.WithContext(ctx)

	s.T().Cleanup(cancel)
	errs.Go(func() error {
		_, err := cmd.ExecuteContextC(ctx)
		if returnError {
			return err
		}
		s.Require().NoError(err)
		s.NoError(err)
		return nil
	})

	ti := time.NewTicker(TickerIncrements)
	defer ti.Stop()
	for {
		select {
		case <-ctx.Done():
			if returnError {
				return true, 0, errs.Wait()
			}
			s.FailNow("Server did not start in time")
		case <-ti.C:
			livezText, statusCode, _ := CurlEndpoint(ctx, fmt.Sprintf("http://127.0.0.1:%d/api/v1/livez", port))
			if string(livezText) == "OK" && statusCode == http.StatusOK {
				return true, port, nil
			}
		}
	}
}

func CurlEndpoint(ctx context.Context, URL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	if err != nil {
		return nil, http.StatusServiceUnavailable, err
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "test", resp.Body)

	responseText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return responseText, resp.StatusCode, nil
}
