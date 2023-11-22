package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"bacalhau-exec-skeleton/handler"
	"bacalhau-exec-skeleton/proto"

	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Expect to be able to find two environment variables when we are launched.
	// Each of them will point at a file which will be used as the socket either
	// for incoming requests (BACALHAU_EXECUTOR_SOCKET) or for reporting state to
	// the compute service (BACALHAU_SUPERVISOR_SOCKET). Neither will have a default
	// value.
	//
	// BACALHAU_SUPERVISOR_SOCKET=
	// BACALHAU_EXECUTOR_SOCKET=

	var handshakeErrs *multierror.Error
	supervisorSocket, found := os.LookupEnv("BACALHAU_SUPERVISOR_SOCKET")
	if !found {
		handshakeErrs = multierror.Append(handshakeErrs, errors.New("was not given BACALHAU_SUPERVISOR_SOCKET for supervisor"))
	}

	executorSocket, found := os.LookupEnv("BACALHAU_EXECUTOR_SOCKET")
	if !found {
		handshakeErrs = multierror.Append(handshakeErrs, errors.New("was not given BACALHAU_EXECUTOR_SOCKET to provide service"))
	}

	err := handshakeErrs.ErrorOrNil()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start: %s\n", err.Error())
		return
	}

	// Create a client connect for talking to the supervisor, whose domain socket
	// we'll have just been given during handshake.
	conn, err := clientConnection(supervisorSocket)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Create our client for talking to the supervisor, and the executor
	// service for running jobs
	client := proto.NewSupervisorClient(conn)
	executorService := handler.NewSkeletonHandler(client)

	// Set up the executor service on the provided socket and using the
	// newly created handler
	listener, err := net.Listen("unix", executorSocket)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterExecutorServer(grpcServer, executorService)
	grpcServer.Serve(listener)
}

func clientConnection(supervisorSocket string) (*grpc.ClientConn, error) {
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", addr)
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithContextDialer(dialer),
	}
	return grpc.Dial(supervisorSocket, options...)
}
