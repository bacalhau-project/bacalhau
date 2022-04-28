package jsonrpc

import (
	"fmt"
	"net/rpc"
)

func JsonRpcMethod(
	host string,
	port int,
	method string,
	req, res interface{},
) error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("Error in dialing. %s", err)
	}
	return client.Call(fmt.Sprintf("JSONRpcServer.%s", method), req, res)
}
