package ipfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

const BACALHAU_LOGFILE = "/tmp/bacalhau.log"

// TODO: We should inject the ipfs binary in the future
func IpfsCommand(repoPath string, args []string) (string, error) {
	log.Trace().Msgf("ipfs command -->   IPFS_PATH=%s ipfs %s\n", repoPath, strings.Join(args, " "))

	// TODO: We should have a struct that allows us to set the ipfs binary, rather than relying on system paths, etc
	ipfs_binary, err := exec.LookPath("ipfs")

	if err != nil {
		log.Error().Msg("Could not find 'ipfs' binary on your path.")
	}

	ipfs_binary_full_path, _ := filepath.Abs(ipfs_binary)

	if strings.Contains(ipfs_binary_full_path, "/snap/") {
		log.Error().Msg("You installed 'ipfs' using snap, which bacalhau is not compatible with. Please install from dist.ipfs.io or directly from your package provider.")
	}

	if repoPath == "" {
		// production mode
		return system.RunCommandGetResults(ipfs_binary_full_path, args)
	} else {
		// dev mode (multiple ipfs servers on the same machine using private local ports)
		return system.RunCommandGetResultsEnv(ipfs_binary_full_path, args, []string{
			"IPFS_PATH=" + repoPath,
		})
	}
}

func StartDaemon(
	ctx context.Context,
	repoPath string,
	ipfsGatewayPort, ipfsApiPort, ipfsSwarmPort int,
) error {
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
	_, err = IpfsCommand(repoPath, []string{
		"config",
		"Addresses.Swarm",
		"--json",
		fmt.Sprintf(`["/ip4/0.0.0.0/tcp/%d"]`, ipfsSwarmPort),
	})
	if err != nil {
		return err
	}
	// we don't want to discover local peers on our network
	// especially when testing in parallel with multiple clusters
	_, err = IpfsCommand(repoPath, []string{
		"config",
		"Discovery.MDNS.Enabled",
		"--json",
		"false",
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("Starting IPFS Daemon: IPFS_PATH=%s ipfs daemon", repoPath)
	cmd := exec.Command("ipfs", "daemon")
	cmd.Env = []string{
		"IPFS_PATH=" + repoPath,
	}

	logfile, err := os.OpenFile(BACALHAU_LOGFILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	cmd.Stderr = logfile
	cmd.Stdout = logfile

	// XXX DANGER WILL ROBINSON: Do not uncomment the following lines or you will get TERRIBLE DEADLOCKS
	// See: https://github.com/golang/go/issues/24050, https://github.com/golang/go/issues/28039
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout

	err = cmd.Start() // nolint
	go func(ctx context.Context, cmd *exec.Cmd) {
		log.Debug().Msg("waiting for ipfs context done\n")
		<-ctx.Done()
		_ = cmd.Process.Kill()
		log.Debug().Msg("got to after closing ipfs daemon\n")
	}(ctx, cmd)
	return nil
}

// this is useful for when developing locally and you want an IPFS server that can co-exist with
// others on the same machine
// TODO: how we connect to ipfs **should** be over the network (rather than shelling out to the ipfs cli with env vars set)
func StartBacalhauDevelopmentIpfsServer(ctx context.Context, connectToMultiAddress string) (string, []string, error) {
	repoDir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create temporary directory for ipfs repo.")
	}

	gatewayPort, err := freeport.GetFreePort()
	if err != nil {
		return "", []string{}, err
	}
	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return "", []string{}, err
	}
	swarmPort, err := freeport.GetFreePort()
	if err != nil {
		return "", []string{}, err
	}
	result, err := IpfsCommand(repoDir, []string{
		"init",
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("Error in command:\noutput: %s\nerror: %s", result, err)
	}
	result, err = IpfsCommand(repoDir, []string{
		"bootstrap", "rm", "--all",
	})
	if err != nil {
		return "", []string{}, fmt.Errorf("Error in command:\noutput: %s\nerror: %s", result, err)
	}
	if connectToMultiAddress != "" {
		result, err = IpfsCommand(repoDir, []string{
			"bootstrap", "add", connectToMultiAddress,
		})
		if err != nil {
			return "", []string{}, fmt.Errorf("Error in command:\noutput: %s\nerror: %s", result, err)
		}
	}

	// TODO: Is this shutting down after a single run?
	err = StartDaemon(ctx, repoDir, gatewayPort, apiPort, swarmPort)
	if err != nil {
		return "", []string{}, err
	}

	nodeAddresses := []string{}

	// give the daemon a better chance to win the race over the lockfile (not perfect though)
	time.Sleep(1 * time.Second)
	err = system.TryUntilSucceedsN(func() error {
		jsonBlob, err := IpfsCommand(repoDir, []string{
			"id",
		})
		if err != nil {
			log.Error().Msgf("error running command: %s\n", err)
			return err
		}
		result := struct {
			Addresses []string
		}{}
		err = json.Unmarshal([]byte(jsonBlob), &result)
		if err != nil {
			log.Error().Msgf("error parsing JSON: %s\n", err)
			return err
		}

		nodeAddresses = result.Addresses

		return nil

	}, "extracting ipfs node id", 10)

	if err != nil {
		return "", []string{}, err
	}

	return repoDir, nodeAddresses, nil
}

func HasCid(repoPath, cid string) (bool, error) {
	log.Debug().Msg("Beginning to collect all refs in IPFS Repo.")
	log.Debug().Msgf("RepoPath {%s}", repoPath)
	allLocalRefString, err := IpfsCommand(repoPath, []string{
		"refs",
		"local",
	})

	log.Debug().Msg("Finished collecting refs in IPFS Repo.")
	if err != nil {
		return false, err
	}
	log.Debug().Msgf("Comparing CID (%s) collecting to all refs in repo.", cid)
	allLocalRefsArray := strings.Split(allLocalRefString, "\n")
	log.Debug().Msgf("Total number of local refs: %d", len(allLocalRefsArray))

	if err != nil {
		return false, err
	}
	got := contains(strings.Split(allLocalRefString, "\n"), cid)
	log.Debug().Msgf("CID (%s) in local refs: %t", cid, got)

	return got, nil
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

func ResultsFolderLogging(nodeId string, jobId string) {
	log.Warn().Msgf("Results folder: ")
}
