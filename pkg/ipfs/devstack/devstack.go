package ipfs_devstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	ipfs_cli "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type IPFSDevServer struct {
	Ctx         context.Context
	Id          string
	Repo        string
	LogFile     string
	Cli         *ipfs_cli.IPFSCli
	GatewayPort int
	ApiPort     int
	SwarmPort   int
}

func NewDevServer(
	ctx context.Context,
) (*IPFSDevServer, error) {
	repoDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack")
	if err != nil {
		return nil, fmt.Errorf("Could not create temporary directory for ipfs repo: %s", err.Error())
	}
	logFile, err := ioutil.TempFile("", "bacalhau-ipfs-devstack")
	if err != nil {
		return nil, fmt.Errorf("Could not create log file for ipfs repo: %s", err.Error())
	}
	gatewayPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("Could not create random port for gateway: %s", err.Error())
	}
	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("Could not create random port for api: %s", err.Error())
	}
	swarmPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("Could not create random port for swarm: %s", err.Error())
	}
	server := &IPFSDevServer{
		Ctx:         ctx,
		Repo:        repoDir,
		LogFile:     logFile.Name(),
		Cli:         ipfs_cli.NewIPFSCli(repoDir),
		GatewayPort: gatewayPort,
		ApiPort:     apiPort,
		SwarmPort:   swarmPort,
	}
	return server, nil
}

func (server *IPFSDevServer) Start(connectToAddress string) error {

	_, err := server.Cli.Run([]string{
		"init",
	})
	if err != nil {
		return err
	}

	// this must be called after init because we need the keys generated
	jsonBlob, err := server.Cli.Run([]string{
		"id",
	})
	if err != nil {
		return err
	}
	result := struct {
		ID string
	}{}
	err = json.Unmarshal([]byte(jsonBlob), &result)
	if err != nil {
		return err
	}

	server.Id = result.ID

	_, err = server.Cli.Run([]string{
		"bootstrap", "rm", "--all",
	})
	if err != nil {
		return err
	}

	if connectToAddress != "" {
		_, err = server.Cli.Run([]string{
			"bootstrap", "add", connectToAddress,
		})
		if err != nil {
			return err
		}
	}

	_, err = server.Cli.Run([]string{
		"config",
		"Addresses.Gateway",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", server.GatewayPort),
	})
	if err != nil {
		return err
	}
	_, err = server.Cli.Run([]string{
		"config",
		"Addresses.API",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", server.ApiPort),
	})
	if err != nil {
		return err
	}
	_, err = server.Cli.Run([]string{
		"config",
		"Addresses.Swarm",
		"--json",
		fmt.Sprintf(`["/ip4/0.0.0.0/tcp/%d"]`, server.SwarmPort),
	})
	if err != nil {
		return err
	}
	_, err = server.Cli.Run([]string{
		"config",
		"Discovery.MDNS.Enabled",
		"--json",
		"false",
	})
	if err != nil {
		return err
	}

	log.Debug().Msgf("Starting IPFS Daemon: IPFS_PATH=%s ipfs daemon", server.Repo)
	cmd := exec.Command("ipfs", "daemon")
	cmd.Env = []string{
		"IPFS_PATH=" + server.Repo,
	}

	logfile, err := os.OpenFile(server.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	cmd.Stderr = logfile
	cmd.Stdout = logfile
	err = cmd.Start() // nolint
	if err != nil {
		return err
	}

	go func(ctx context.Context, cmd *exec.Cmd) {
		<-ctx.Done()
		log.Debug().Msg("Closing ipfs server")
		_ = cmd.Process.Kill()
		log.Debug().Msg("Closed ipfs server")
	}(server.Ctx, cmd)

	return nil
}

func (server *IPFSDevServer) Address() string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", server.SwarmPort, server.Id)
}
