package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type DevStackNodeIPFS struct {
	IpfsNode   *ipfs.Node
	IpfsClient *ipfs.Client
}

type DevStackIPFS struct {
	Nodes          []*DevStackNodeIPFS
	CleanupManager *system.CleanupManager
}

// a devstack but with only IPFS servers connected to each other
func NewDevStackIPFS(cm *system.CleanupManager, count int) (*DevStackIPFS, error) {
	nodes := []*DevStackNodeIPFS{}
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////
		var err error
		var ipfsSwarmAddrs []string
		if i > 0 {
			ipfsSwarmAddrs, err = nodes[0].IpfsNode.SwarmAddresses()
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
		}

		ipfsNode, err := ipfs.NewLocalNode(cm, ipfsSwarmAddrs)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}

		ipfsClient, err := ipfsNode.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs client: %w", err)
		}

		devStackNode := &DevStackNodeIPFS{
			IpfsNode:   ipfsNode,
			IpfsClient: ipfsClient,
		}

		nodes = append(nodes, devStackNode)
	}

	stack := &DevStackIPFS{
		Nodes:          nodes,
		CleanupManager: cm,
	}

	return stack, nil
}

func (stack *DevStackIPFS) PrintNodeInfo() {
	logString := `
-------------------------------
ipfs
-------------------------------

command="add -q testdata/grep_file.txt"
	`
	for _, node := range stack.Nodes {
		logString += fmt.Sprintf(`
cid=$(IPFS_PATH=%s ipfs $command)
curl http://127.0.0.1:%d/api/v0/id`, node.IpfsNode.RepoPath, node.IpfsNode.APIPort)
	}

	log.Trace().Msg(logString + "\n")
}

func (stack *DevStackIPFS) addItemToNodes(nodeCount int, filePath string, isDirectory bool) (string, error) {
	var res string
	for i, node := range stack.Nodes {
		if node == nil {
			continue
		}
		if i >= nodeCount {
			continue
		}

		cid, err := node.IpfsClient.Put(context.Background(), filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to node %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs node '%s'", cid, node.IpfsNode.ID())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func (stack *DevStackIPFS) AddFileToNodes(nodeCount int, filePath string) (string, error) {
	return stack.addItemToNodes(nodeCount, filePath, false)
}

func (stack *DevStackIPFS) AddFolderToNodes(nodeCount int, folderPath string) (string, error) {
	return stack.addItemToNodes(nodeCount, folderPath, true)
}

func (stack *DevStackIPFS) AddTextToNodes(nodeCount int, fileContent []byte) (string, error) {
	testDir, err := ioutil.TempDir("", "bacalhau-test")

	if err != nil {
		return "", err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)

	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(nodeCount, testFilePath)
}
