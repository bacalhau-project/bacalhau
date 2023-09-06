package ipfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func AddFileToNodes(ctx context.Context, filePath string, clients ...Client) (string, error) {
	var res string
	for i, client := range clients {
		cid, err := client.Put(ctx, filePath)
		if err != nil {
			return "", errors.Wrap(err, fmt.Sprintf("error adding %q to node %d", filePath, i))
		}

		log.Ctx(ctx).Debug().Msgf("Added CID %q to IPFS node %q", cid, client.APIAddress())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func AddTextToNodes(ctx context.Context, fileContent []byte, clients ...Client) (string, error) {
	tempDir, err := os.MkdirTemp("", "bacalhau-test")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return "", err
	}

	return AddFileToNodes(ctx, testFilePath, clients...)
}

func ParsePeersString(peers []string) ([]peer.AddrInfo, error) {
	// Parse the bootstrap node multiaddrs and fetch their IPFS peer info:
	var res []peer.AddrInfo
	for _, p := range peers {
		if p == "" {
			continue
		}
		pi, err := peer.AddrInfoFromString(p)
		if err != nil {
			return nil, err
		}
		res = append(res, *pi)
	}

	return res, nil
}

func copyFile(sourcePath, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError(sourcePath, sourceFile)

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError(destinationPath, destinationFile)

	copied, err := io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	} else if copied < sourceInfo.Size() {
		return fmt.Errorf("expected to copy %d bytes but only copied %d", sourceInfo.Size(), copied)
	}

	if err := destinationFile.Chmod(sourceInfo.Mode()); err != nil {
		return err
	}

	return nil
}
