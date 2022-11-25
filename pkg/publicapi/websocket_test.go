//go:build unit || !integration

package publicapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type WebsocketSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestWebsocketSuite(t *testing.T) {
	suite.Run(t, new(WebsocketSuite))
}

// Before each test
func (s *WebsocketSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

func (s *WebsocketSuite) TestWebsocketEverything() {
	ctx := context.Background()
	// hairpin on as this depends on seeing our own events
	c, cm := SetupRequesterNodeForTests(s.T(), true)
	defer cm.Cleanup()
	// string.Replace http with ws in c.BaseURI
	url := "ws" + c.BaseURI[4:] + "/websocket"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(s.T(), err)

	eventChan := make(chan model.JobEvent)
	go func() {
		for {
			var event model.JobEvent
			err = conn.ReadJSON(&event)
			require.NoError(s.T(), err)
			eventChan <- event
		}
	}()

	genericJob := MakeGenericJob()
	_, err = c.Submit(ctx, genericJob, nil)
	require.NoError(s.T(), err)

	event := <-eventChan
	require.Equal(s.T(), event.EventName.String(), "Created")

}

func (s *WebsocketSuite) TestWebsocketSingleJob() {
	ctx := context.Background()
	c, cm := SetupRequesterNodeForTests(s.T(), true)
	defer cm.Cleanup()

	genericJob := MakeGenericJob()
	j, err := c.Submit(ctx, genericJob, nil)
	require.NoError(s.T(), err)

	url := "ws" + c.BaseURI[4:] + fmt.Sprintf("/websocket?job_id=%s", j.ID)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(s.T(), err)

	var event model.JobEvent
	err = conn.ReadJSON(&event)
	require.NoError(s.T(), err)
	require.Equal(s.T(), event.EventName.String(), "Created")
}
