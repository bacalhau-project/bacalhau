package proxy

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
)

// sendResponse marshals the response and sends it back to the requester.
func sendResponse[Response any](conn *nats.Conn, reply string, result *concurrency.AsyncResult[Response]) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error encoding %T: %s", result.Value, err)
	}

	return conn.Publish(reply, resultData)
}
