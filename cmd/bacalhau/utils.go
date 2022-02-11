package bacalhau

import (
	"fmt"
	"net/rpc"
	"strings"
)

var listOutputFormat string
var tableOutputWide bool

func JsonRpcMethod(method string, req, res interface{}) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", jsonrpcHost, jsonrpcPort))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JobServer.%s", method), req, res)
}

func getString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < 20 {
		return st
	}

	return st[:20] + "..."
}

func shortId(id string) string {
	parts := strings.Split(id, "-")
	return parts[0]
}
