package internal

import (
	"fmt"
	"log"
	"net/http"

	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type ListArgs struct {
}

type SubmitArgs struct {
	Job *types.Job
}

func (server *JobServer) List(r *http.Request, args *ListArgs, reply *types.ListResponse) error {
	*reply = types.ListResponse{
		Jobs:       server.ComputeNode.Jobs,
		JobState:   server.ComputeNode.JobState,
		JobStatus:  server.ComputeNode.JobStatus,
		JobResults: server.ComputeNode.JobResults,
	}
	return nil
}

func (server *JobServer) Submit(r *http.Request, args *SubmitArgs, reply *types.Job) error {
	server.ComputeNode.Publish(args.Job)
	*reply = *args.Job
	return nil
}

// func RunBacalhauRpcServer(host string, port int, computeNode *ComputeNode) {
// 	job := &JobServer{
// 		ComputeNode: computeNode,
// 	}
// 	server := rpc.NewServer()
// 	err := server.Register(job)
// 	if err != nil {
// 		log.Fatalf("Format of service Job isn't correct. %s", err)
// 	}
// 	server.HandleHTTP("/", "/debug")
// 	//rpc.HandleHTTP()
// 	l, e := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
// 	if e != nil {
// 		log.Fatalf("Couldn't start listening on port %d. Error %s", port, e)
// 	}
// 	log.Println("Serving RPC handler")
// 	err = http.Serve(l, nil)
// 	if err != nil {
// 		log.Fatalf("Error serving: %s", err)
// 	}
// }

func RunBacalhauJsonRpcServer(host string, port int, computeNode *ComputeNode) {
	job := &JobServer{
		ComputeNode: computeNode,
	}

	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	err := server.RegisterService(job, "")
	if err != nil {
		log.Fatalf("server.RegisterService failed: %s", err)
	}
	router := mux.NewRouter()
	router.Handle("/", server)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), router)
	if err != nil {
		log.Fatalf("http.ListenAndServe failed: %s", err)
	}
}
