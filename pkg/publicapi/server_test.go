//go:build unit || !integration

package publicapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIServerTestSuite struct {
	suite.Suite
	server *Server
}

func (suite *APIServerTestSuite) SetupTest() {
	params := ServerParams{
		Router:  echo.New(),
		Address: "localhost",
		Port:    8080,
		HostID:  "testHostID",
		Config:  *NewConfig(),
	}
	var err error
	suite.server, err = NewAPIServer(params)
	assert.NoError(suite.T(), err)
}

func (suite *APIServerTestSuite) TestNewAPIServer() {
	assert.NotNil(suite.T(), suite.server)
}

func (suite *APIServerTestSuite) TestGetURI() {
	uri := suite.server.GetURI()
	assert.NotNil(suite.T(), uri)
	assert.Equal(suite.T(), "http", uri.Scheme)
	assert.Equal(suite.T(), "localhost", uri.Hostname())
	assert.NotEqual(suite.T(), 8080, uri.Port())
}

func (suite *APIServerTestSuite) TestListenAndServe() {
	ctx := context.Background()
	go func() {
		err := suite.server.ListenAndServe(ctx)
		assert.NoError(suite.T(), err)
	}()

	time.Sleep(1 * time.Second) // give it some time to start

	// Make a request to ensure the server is running
	resp, err := http.Get(suite.server.GetURI().String())
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	resp.Body.Close()
}

func (suite *APIServerTestSuite) TestShutdown() {
	ctx := context.Background()
	err := suite.server.Shutdown(ctx)
	assert.NoError(suite.T(), err)
}

func TestAPIServerTestSuite(t *testing.T) {
	suite.Run(t, new(APIServerTestSuite))
}
