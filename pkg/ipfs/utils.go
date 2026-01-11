package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
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
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFilePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW|util.OS_ALL_R)
	if err != nil {
		return "", err
	}

	return AddFileToNodes(ctx, testFilePath, clients...)
}

func SortLocalhostFirst(multiAddresses []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	multiAddresses = slices.Clone(multiAddresses)
	preferLocalhost := func(m multiaddr.Multiaddr) int {
		count := 0
		if _, err := m.ValueForProtocol(multiaddr.P_TCP); err == nil {
			count++
		}
		if ip, err := m.ValueForProtocol(multiaddr.P_IP4); err == nil {
			count++
			if ip == "127.0.0.1" {
				count++
			}
		} else if ip, err := m.ValueForProtocol(multiaddr.P_IP6); err == nil && ip != "::1" {
			count++
		}
		return count
	}
	sort.Slice(multiAddresses, func(i, j int) bool {
		return preferLocalhost(multiAddresses[i]) > preferLocalhost(multiAddresses[j])
	})

	return multiAddresses
}
