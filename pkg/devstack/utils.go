package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"

	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/rs/zerolog/log"
)

const TmpFileName = "tempfile.txt"

func ToIPFSClients(nodes []*node.Node) []*ipfs.Client {
	res := []*ipfs.Client{}
	for _, n := range nodes {
		res = append(res, n.IPFSClient)
	}

	return res
}

func AddFileToNodesForTests(ctx context.Context, filePath string, clients ...*ipfs.Client) (string, error) {
	var res string
	for i, client := range clients {
		cid, err := client.Put(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to n %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs n '%s'", cid, client.APIAddress())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func AddTextToNodesForTests(ctx context.Context, fileContent []byte, clients ...*ipfs.Client) (string, error) {
	tmpDir, err := ioutil.TempDir("", "bacalhau-add-text-to-nodes")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	tmpFile := tmpDir + "/" + TmpFileName

	err = os.WriteFile(tmpFile, fileContent, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return "", err
	}

	return AddFileToNodesForTests(ctx, tmpFile, clients...)
}
