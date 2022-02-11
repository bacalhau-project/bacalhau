package ipfs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/filecoin-project/bacalhau/internal/system"
)

const IPFS_REPO_LOCATION string = "data/ipfs"

func GetIpfsRepo(hostId string) string {
	return fmt.Sprintf("%s/%s", IPFS_REPO_LOCATION, hostId)
}

func EnsureIpfsRepo(hostId string) (string, error) {
	folder := GetIpfsRepo(hostId)
	err := system.RunCommand("mkdir", []string{
		"-p",
		folder,
	})
	return folder, err
}

func IpfsCommand(repoPath string, args []string) (string, error) {
	return system.RunCommandGetResultsEnv("ipfs", args, []string{
		"IPFS_PATH=" + repoPath,
	})
}

func Init(repoPath string) error {
	_, err := IpfsCommand(repoPath, []string{
		"init",
	})
	return err
}

func StartDaemon(repoPath string, ipfsGatewayPort, ipfsApiPort int) error {
	_, err := IpfsCommand(repoPath, []string{
		"config",
		"Addresses.Gateway",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", ipfsGatewayPort),
	})
	if err != nil {
		return err
	}
	_, err = IpfsCommand(repoPath, []string{
		"config",
		"Addresses.API",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", ipfsApiPort),
	})
	if err != nil {
		return err
	}
	go func() {
		cmd := exec.Command("ipfs", "daemon")
		cmd.Env = []string{
			"IPFS_PATH=" + repoPath,
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Run()
	}()
	return nil
}

func HasCid(repoPath, cid string) (bool, error) {
	allLocalRefString, err := IpfsCommand(repoPath, []string{
		"refs",
		"local",
	})
	if err != nil {
		return false, err
	}
	return contains(strings.Split(allLocalRefString, "\n"), cid), nil
}

func AddFolder(repoPath, folder string) (string, error) {
	allCidsString, err := IpfsCommand(repoPath, []string{
		"add",
		"-rq",
		folder,
	})
	if err != nil {
		return "", err
	}
	allCids := strings.Split(allCidsString, "\n")
	// -2 is because it's the second last one before the newline
	return allCids[len(allCids)-2], nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
