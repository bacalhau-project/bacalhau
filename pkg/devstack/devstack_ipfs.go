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

type DevStackIPFS struct {
	IPFSClients    []*ipfs.Client
	CleanupManager *system.CleanupManager
}

// A devstack but with only IPFS servers connected to each other
func NewDevStackIPFS(ctx context.Context, cm *system.CleanupManager, count int) (*DevStackIPFS, error) {
	clients := []*ipfs.Client{}
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////
		var err error
		var ipfsSwarmAddrs []string
		if i > 0 {
			ipfsSwarmAddrs, err = clients[0].SwarmAddresses(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
		}

		ipfsNode, err := ipfs.NewLocalNode(ctx, cm, ipfsSwarmAddrs)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}

		ipfsClient, err := ipfsNode.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs client: %w", err)
		}

		clients = append(clients, ipfsClient)
	}

	stack := &DevStackIPFS{
		IPFSClients:    clients,
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
	for _, node := range stack.IPFSClients {
		logString += fmt.Sprintf(`
cid=$(ipfs --api %s ipfs $command)
curl -XPOST %s`, node.APIAddress(), node.APIAddress())
	}

	log.Trace().Msg(logString + "\n")
}

func (stack *DevStackIPFS) addItemToNodes(ctx context.Context, nodeCount int, filePath string, isDirectory bool) (string, error) {
	var res string
	for i, node := range stack.IPFSClients {
		if node == nil {
			continue
		}
		if i >= nodeCount {
			continue
		}

		cid, err := node.Put(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to node %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs node '%s'", cid, node.APIAddress())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func (stack *DevStackIPFS) AddFileToNodes(ctx context.Context, nodeCount int, filePath string) (string, error) {
	return stack.addItemToNodes(ctx, nodeCount, filePath, false)
}

func (stack *DevStackIPFS) AddFolderToNodes(ctx context.Context, nodeCount int, folderPath string) (string, error) {
	return stack.addItemToNodes(ctx, nodeCount, folderPath, true)
}

func (stack *DevStackIPFS) AddTextToNodes(ctx context.Context, nodeCount int, fileContent []byte) (string, error) {
	testDir, err := ioutil.TempDir("", "bacalhau-test")

	if err != nil {
		return "", err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)

	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(ctx, nodeCount, testFilePath)
}
