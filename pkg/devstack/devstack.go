package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	ipfs_cli "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	ipfs_devstack "github.com/filecoin-project/bacalhau/pkg/ipfs/devstack"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type DevStackNode struct {
	ComputeNode   *compute_node.ComputeNode
	RequesterNode *requestor_node.RequesterNode
	IpfsNode      *ipfs_devstack.IPFSDevServer
	IpfsCli       *ipfs_cli.IPFSCli
	Transport     *libp2p.Transport

	ApiServer *publicapi.APIServer
}

type DevStack struct {
	Nodes []*DevStackNode
}

type GetExecutorsFunc func(ipfsMultiAddress string, nodeIndex int) (
	map[string]executor.Executor, error)

type GetVerifiersFunc func(ipfsMultiAddress string, nodeIndex int) (
	map[string]verifier.Verifier, error)

func NewDevStack(
	cm *system.CleanupManager,
	count, badActors int,
	getExecutors GetExecutorsFunc,
	getVerifiers GetVerifiersFunc,
	jobSelectionPolicy compute_node.JobSelectionPolicy,
) (
	*DevStack, error) {

	ctx, span := newSpan("NewDevStack")
	defer span.End()

	nodes := []*DevStackNode{}
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////
		ipfsConnectAddress := ""

		if i > 0 {
			// connect the libp2p scheduler node
			firstNode := nodes[0]
			ipfsConnectAddress = firstNode.IpfsNode.SwarmAddress()
		}

		// construct the ipfs, scheduler, requester, compute and jsonRpc nodes
		ipfsNode, err := ipfs_devstack.NewDevServer(cm, true)
		if err != nil {
			return nil, fmt.Errorf(
				"devstack: failed to create ipfs node: %w", err)
		}

		err = ipfsNode.Start(ipfsConnectAddress)
		if err != nil {
			return nil, fmt.Errorf(
				"devstack: failed to start ipfs node: %w", err)
		}

		log.Debug().Msgf("IPFS dev server started: %s", ipfsNode.ApiAddress())

		//////////////////////////////////////
		// Scheduler
		//////////////////////////////////////
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		transport, err := libp2p.NewTransport(cm, libp2pPort)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Requestor node
		//////////////////////////////////////
		requesterNode, err := requestor_node.NewRequesterNode(transport)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Compute node
		//////////////////////////////////////
		executors, err := getExecutors(ipfsNode.ApiAddress(), i)
		if err != nil {
			return nil, err
		}

		verifiers, err := getVerifiers(ipfsNode.ApiAddress(), i)
		if err != nil {
			return nil, err
		}

		computeNode, err := compute_node.NewComputeNode(
			transport,
			executors,
			verifiers,
			jobSelectionPolicy,
		)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// JSON RPC
		//////////////////////////////////////
		apiPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		apiServer := publicapi.NewServer(requesterNode, "0.0.0.0", apiPort)
		go func(ctx context.Context) {
			if err := apiServer.ListenAndServe(ctx, cm); err != nil {
				panic(err) // if api server can't run, devstack should stop
			}
		}(context.Background())

		log.Debug().Msgf("public API server started: 0.0.0.0:%d", apiPort)

		//////////////////////////////////////
		// intra-connections
		//////////////////////////////////////

		go func(ctx context.Context) {
			if err = transport.Start(ctx); err != nil {
				panic(err) // if transport can't run, devstack should stop
			}
		}(context.Background())

		log.Debug().Msgf("libp2p server started: %d", libp2pPort)

		if len(nodes) > 0 {
			// connect the libp2p scheduler node
			firstNode := nodes[0]

			// get the libp2p id of the first scheduler node
			libp2pHostId, err := firstNode.Transport.HostID(ctx)
			if err != nil {
				return nil, err
			}

			// connect this scheduler to the first
			firstSchedulerAddress := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", firstNode.Transport.Port, libp2pHostId)
			log.Debug().Msgf("Connect to first libp2p scheduler node: %s", firstSchedulerAddress)
			err = transport.Connect(ctx, firstSchedulerAddress)
			if err != nil {
				return nil, err
			}
		}

		devStackNode := &DevStackNode{
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			IpfsNode:      ipfsNode,
			IpfsCli:       ipfs_cli.NewIPFSCli(ipfsNode.Repo),
			Transport:     transport,
			ApiServer:     apiServer,
		}

		nodes = append(nodes, devStackNode)
	}

	stack := &DevStack{
		Nodes: nodes,
	}

	return stack, nil
}

func (stack *DevStack) PrintNodeInfo() {
	logString := ""

	for nodeIndex, node := range stack.Nodes {

		logString = logString + fmt.Sprintf(`
export IPFS_PATH_%d=%s
export API_PORT_%d=%d`,
			nodeIndex,
			node.IpfsNode.Repo,
			nodeIndex,
			stack.Nodes[0].ApiServer.Port,
		)

	}

	for nodeIndex, node := range stack.Nodes {

		logString = logString + fmt.Sprintf(`
-------------------------------
node %d
-------------------------------

export IPFS_API_PORT_%d=%d
export IPFS_PATH_%d=%s
export API_PORT_%d=%d
cid=$(IPFS_PATH=%s ipfs add -q testdata/grep_file.txt)
curl -XPOST http://127.0.0.1:%d/api/v0/id
`,
			nodeIndex,
			nodeIndex,
			node.IpfsNode.ApiPort,
			nodeIndex,
			node.IpfsNode.Repo,
			nodeIndex,
			stack.Nodes[0].ApiServer.Port,
			node.IpfsNode.Repo,
			node.IpfsNode.ApiPort,
		)

	}

	log.Info().Msg(logString)
}

func (stack *DevStack) AddFileToNodes(nodeCount int, filePath string) (string, error) {
	returnFileCid := ""

	// ipfs add the file to 2 nodes
	// this tests self selection
	for i, node := range stack.Nodes {
		if i >= nodeCount {
			continue
		}

		nodeId, err := node.ComputeNode.Transport.HostID(context.Background())

		if err != nil {
			return "", err
		}

		fileCid, err := node.IpfsCli.Run([]string{
			"add", "-Q", filePath,
		})

		if err != nil {
			return "", err
		}

		fileCid = strings.TrimSpace(fileCid)
		returnFileCid = fileCid
		log.Debug().Msgf("Added CID: %s to NODE: %s", fileCid, nodeId)
	}

	return returnFileCid, nil
}

func (stack *DevStack) AddTextToNodes(nodeCount int, fileContent []byte) (string, error) {
	testDir, err := ioutil.TempDir("", "bacalhau-test")

	if err != nil {
		return "", err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)
	err = os.WriteFile(testFilePath, fileContent, 0644)
	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(nodeCount, testFilePath)
}

func (stack *DevStack) GetJobStates(ctx context.Context, jobId string) (map[string]types.JobStateType, error) {
	apiClient := publicapi.NewAPIClient(stack.Nodes[0].ApiServer.GetURI())

	job, ok, err := apiClient.Get(ctx, jobId)
	if err != nil {
		return nil, fmt.Errorf(
			"devstack: error fetching job %s: %v", jobId, err)
	}
	if !ok {
		return nil, nil
	}

	states := map[string]types.JobStateType{}
	for id, state := range job.State {
		states[id] = state.State
	}

	return states, nil
}

// a function that is given a map of nodeid -> job states
// and will throw an error if anything about that is wrong
type CheckJobStatesFunction func(map[string]types.JobStateType) (bool, error)

// there should be zero errors with any job
func WaitForJobThrowErrors(errorStates []types.JobStateType) CheckJobStatesFunction {
	return func(jobStates map[string]types.JobStateType) (bool, error) {
		log.Trace().Msgf("WaitForJobThrowErrors:\nerrorStates = %+v,\njobStates = %+v", errorStates, jobStates)
		for id, state := range jobStates {
			if system.StringArrayContains(system.GetJobStateStringArray(errorStates), string(state)) {
				return false, fmt.Errorf("job %s has error state: %s", id, string(state))
			}
		}
		return true, nil
	}
}

// there must be exactly len(nodeIds)
// each state must be the given type
// each seen node id must be present in the presented array
// this is useful for testing (did only the nodes that should have completed the job run it)
func WaitForJobAllHaveState(nodeIds []string, state types.JobStateType) CheckJobStatesFunction {
	return func(jobStates map[string]types.JobStateType) (bool, error) {
		log.Trace().Msgf("WaitForJobShouldHaveStates:\nnodeIds = %+v,\nstate = %s\njobStates = %+v", nodeIds, state, jobStates)
		if len(jobStates) != len(nodeIds) {
			return false, nil
		}
		seenAll := true
		for _, nodeId := range nodeIds {
			seenState, ok := jobStates[nodeId]
			if !ok {
				seenAll = false
			} else if seenState != state {
				seenAll = false
			}
		}
		return seenAll, nil
	}
}

func (stack *DevStack) WaitForJob(
	ctx context.Context,
	jobId string,
	checkJobStateFunctions ...CheckJobStatesFunction,
) error {
	waiter := &system.FunctionWaiter{
		Name:        "wait for job",
		MaxAttempts: 100,
		Delay:       time.Second * 1,
		Handler: func() (bool, error) {
			// load the current states of the job
			states, err := stack.GetJobStates(ctx, jobId)
			if err != nil {
				return false, err
			}
			allOk := true
			for _, checkFunction := range checkJobStateFunctions {
				stepOk, err := checkFunction(states)
				if err != nil {
					return false, err
				}
				if !stepOk {
					allOk = false
				}
			}
			return allOk, nil
		},
	}

	return waiter.Wait()
}

func (stack *DevStack) GetNode(ctx context.Context, nodeId string) (
	*DevStackNode, error) {

	for _, node := range stack.Nodes {
		id, err := node.Transport.HostID(ctx)
		if err != nil {
			return nil, err
		}

		if id == nodeId {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeId)
}

func (stack *DevStack) GetNodeIds() ([]string, error) {
	ids := []string{}
	for _, node := range stack.Nodes {
		id, err := node.Transport.HostID(context.Background())
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (stack *DevStack) GetShortIds() ([]string, error) {
	ids, err := stack.GetNodeIds()
	if err != nil {
		return ids, err
	}
	shortids := []string{}
	for _, id := range ids {
		shortids = append(shortids, system.ShortId(id))
	}
	return shortids, nil
}

func newSpan(name string) (context.Context, trace.Span) {
	return system.Span(context.Background(), "devstack", name)
}
