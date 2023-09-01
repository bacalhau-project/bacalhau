//go:build unit || !integration

package publicapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIServerTestSuite struct {
	suite.Suite
	server *Server
}

func (suite *APIServerTestSuite) SetupTest() {
	params := ServerParams{
		Router:  chi.NewRouter(),
		Address: "localhost",
		Port:    8080,
		HostID:  "testHostID",
		Config:  *NewConfig(),
	}
	var err error
	suite.server, err = NewAPIServer(params)
	assert.NotNil(suite.T(), suite.server)
	assert.NoError(suite.T(), err)
}

func (suite *APIServerTestSuite) TestGetURI() {
	uri := suite.server.GetURI()
	assert.NotNil(suite.T(), uri)
	assert.Equal(suite.T(), "http", uri.Scheme)
	assert.Equal(suite.T(), "localhost", uri.Hostname())
	assert.Equal(suite.T(), "8080", uri.Port())
}

func (suite *APIServerTestSuite) TestListenAndServe() {
	ctx := context.Background()
	go func() {
		err := suite.server.ListenAndServe(ctx)
		assert.NoError(suite.T(), err)
	}()

	suite.Eventually(func() bool {
		resp, err := http.Get(suite.server.GetURI().String())
		defer func() {
			if resp != nil {
				resp.Body.Close()
			}
		}()
		return err == nil && resp != nil && resp.StatusCode == http.StatusNotFound
	}, 1*time.Second, 50*time.Millisecond) // give it some time to start

	// Make a request to ensure the server is running
	resp, err := http.Get(suite.server.GetURI().String())
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	resp.Body.Close()

	// shutdown the server
	err = suite.server.Shutdown(ctx)
	assert.NoError(suite.T(), err)

	// Make a request to ensure the server is shutdown
	resp, err = http.Get(suite.server.GetURI().String())
	assert.Error(suite.T(), err)
}

func (suite *APIServerTestSuite) TestShutdownNotRunning() {
	// shutdown the server when it's not running should not error
	ctx := context.Background()
	err := suite.server.Shutdown(ctx)
	assert.NoError(suite.T(), err)
}

func TestAPIServerTestSuite(t *testing.T) {
	suite.Run(t, new(APIServerTestSuite))
}
