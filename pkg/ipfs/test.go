package ipfs

import (
	"context"
	"testing"
)

// MustHaveIPFS returns the IPFS node address of a local
// instance, or fails the test.
func MustHaveIPFS(t testing.TB) string {
	connection := HasIPFS(t)
	if connection != "" {
		return connection
	}

	t.Skip("Cannot run this test because it requires a local IPFS Node")
	return ""
}

func HasIPFS(t testing.TB) string {
	possibleConnectString := "/ip4/127.0.0.1/tcp/5001"
	client, err := NewClientUsingRemoteHandler(context.Background(), possibleConnectString)
	if err == nil && client != nil {
		return possibleConnectString
	}

	return ""
}
