package bacalhau

import (
	"fmt"
	"net/rpc"
)

func JsonRpcMethod(method string, req, res interface{}) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", jsonrpcHost, jsonrpcPort))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JobServer.%s", method), req, res)
}
