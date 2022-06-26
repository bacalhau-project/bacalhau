package devstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	ipfsCLI "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	ipfsDevstack "github.com/filecoin-project/bacalhau/pkg/ipfs/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type DevStackNodeIPFS struct {
	IpfsNode *ipfsDevstack.IPFSDevServer
	IpfsCLI  *ipfsCLI.IPFSCli
}

type DevStackIPFS struct {
	Nodes          []*DevStackNodeIPFS
	CleanupManager *system.CleanupManager
}

// a devstack but with only IPFS servers connected to each other
func NewDevStackIPFS(cm *system.CleanupManager, count int) (
	*DevStackIPFS, error) {

	nodes := []*DevStackNodeIPFS{}
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
		ipfsNode, err := ipfsDevstack.NewDevServer(cm, true)
		if err != nil {
			return nil, err
		}

		err = ipfsNode.Start(ipfsConnectAddress)
		if err != nil {
			return nil, err
		}

		devStackNode := &DevStackNodeIPFS{
			IpfsNode: ipfsNode,
			IpfsCLI:  ipfsCLI.NewIPFSCli(ipfsNode.Repo),
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

		logString = logString + fmt.Sprintf(`
cid=$(IPFS_PATH=%s ipfs $command)
curl http://127.0.0.1:%d/api/v0/id`, node.IpfsNode.Repo, node.IpfsNode.APIPort)

	}

	log.Info().Msg(logString + "\n")
}

func (stack *DevStackIPFS) addItemToNodes(nodeCount int, filePath string, isDirectory bool) (string, error) {
	returnFileCid := ""

	// ipfs add the file to 2 nodes
	// this tests self selection
	for i, node := range stack.Nodes {
		if i >= nodeCount {
			continue
		}

		args := []string{"add", "-Q"}

		if isDirectory {
			args = append(args, "-r")
		}

		args = append(args, filePath)

		fileCid, err := node.IpfsCLI.Run(args)

		if err != nil {
			return "", err
		}

		fileCid = strings.TrimSpace(fileCid)
		returnFileCid = fileCid
		log.Debug().Msgf("Added CID: %s to NODE: %d", fileCid, i)
	}

	return returnFileCid, nil
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
	err = os.WriteFile(testFilePath, fileContent, 0644)

	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(nodeCount, testFilePath)
}
