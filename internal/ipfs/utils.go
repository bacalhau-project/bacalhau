package ipfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/phayes/freeport"
)

func IpfsCommand(repoPath string, args []string) (string, error) {
	fmt.Printf("ipfs command:\nipfs %s\n", strings.Join(args, " "))
	if repoPath == "" {
		return system.RunCommandGetResults("ipfs", args)
	} else {
		return system.RunCommandGetResultsEnv("ipfs", args, []string{
			"IPFS_PATH=" + repoPath,
		})
	}
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
	fmt.Printf("IPFS_PATH=%s ipfs daemon\n", repoPath)
	go func() {
		cmd := exec.Command("ipfs", "daemon")
		cmd.Env = []string{
			"IPFS_PATH=" + repoPath,
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			fmt.Printf("error running ipfs daemon: %s\n", err)
		}
	}()
	return nil
}

// this is useful for when developing locally and you want an IPFS server that can co-exist with
// others on the same machine
// TODO: how we connect to ipfs **should** be over the network (rather than shelling out to the ipfs cli with env vars set)
func StartBacalhauDevelopmentIpfsServer(connectToMultiAddress string) (string, string, error) {
	repoDir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		log.Fatal(err)
	}
	_, err = system.EnsureSystemDirectory(repoDir)
	if err != nil {
		return "", "", err
	}
	gatewayPort, err := freeport.GetFreePort()
	if err != nil {
		return "", "", err
	}
	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return "", "", err
	}
	_, err = IpfsCommand(repoDir, []string{
		"init",
	})
	if err != nil {
		return "", "", err
	}
	_, err = IpfsCommand(repoDir, []string{
		"bootstrap", "rm", "--all",
	})
	if err != nil {
		return "", "", err
	}
	if connectToMultiAddress != "" {
		_, err = IpfsCommand(repoDir, []string{
			"bootstrap", "add", connectToMultiAddress,
		})
	}
	err = StartDaemon(repoDir, gatewayPort, apiPort)
	if err != nil {
		return "", "", err
	}

	nodeAddress := ""

	// give the daemon a better chance to win the race over the lockfile (not perfect though)
	time.Sleep(1 * time.Second)
	err = system.TryUntilSucceedsN(func() error {
		jsonBlob, err := IpfsCommand(repoDir, []string{
			"id",
		})
		if err != nil {
			fmt.Printf("error running command: %s\n", err)
			return err
		}
		result := struct {
			Addresses []string
		}{}
		err = json.Unmarshal([]byte(jsonBlob), &result)
		if err != nil {
			fmt.Printf("error parsing JSON: %s\n", err)
			return err
		}
		if len(result.Addresses) > 0 {
			nodeAddress = result.Addresses[0]
			return nil
		} else {
			return fmt.Errorf("no node address")
		}
	}, "extracting ipfs node id", 10)

	if err != nil {
		return "", "", err
	}

	return repoDir, nodeAddress, nil
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
