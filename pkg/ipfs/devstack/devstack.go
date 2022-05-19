package ipfs_devstack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	ipfs_cli "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type IPFSDevServer struct {
	cancelContext *system.CancelContext
	Id            string
	Repo          string
	LogFile       string
	Isolated      bool
	Cli           *ipfs_cli.IPFSCli
	GatewayPort   int
	ApiPort       int
	SwarmPort     int
}

func NewDevServer(
	cancelContext *system.CancelContext,
	isolated bool,
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
	cli := ipfs_cli.NewIPFSCli(repoDir)
	_, err = cli.Run([]string{
		"init",
		"--profile",
		"test",
	})
	if err != nil {
		return nil, err
	}

	// this must be called after init because we need the keys generated
	jsonBlob, err := cli.Run([]string{
		"id",
	})
	if err != nil {
		return nil, err
	}
	idResult := struct {
		ID string
	}{}
	err = json.Unmarshal([]byte(jsonBlob), &idResult)
	if err != nil {
		return nil, err
	}

	server := &IPFSDevServer{
		cancelContext: cancelContext,
		Id:            idResult.ID,
		Repo:          repoDir,
		LogFile:       logFile.Name(),
		Cli:           cli,
		Isolated:      isolated,
		GatewayPort:   gatewayPort,
		ApiPort:       apiPort,
		SwarmPort:     swarmPort,
	}
	return server, nil
}

func (server *IPFSDevServer) Start(connectToAddress string) error {

	if server.Isolated {
		_, err := server.Cli.Run([]string{
			"bootstrap", "rm", "--all",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"AutoNAT.ServiceMode",
			"disabled",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.EnableHolePunching",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.DisableNatPortMap",
			"--bool",
			"true",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.RelayClient.Enabled",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.RelayService.Enabled",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.Transports.Network.Relay",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}
		_, err = server.Cli.Run([]string{
			"config",
			"Swarm.Transports.Network.Relay",
			"--json",
			"false",
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
		_, err = server.Cli.Run([]string{
			"config",
			"Addresses.Announce",
			"--json",
			fmt.Sprintf(`["/ip4/127.0.0.1/tcp/%d"]`, server.SwarmPort),
		})
		if err != nil {
			return err
		}
	}
	_, err := server.Cli.Run([]string{
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

	if connectToAddress != "" {
		_, err := server.Cli.Run([]string{
			"bootstrap", "add", connectToAddress,
		})
		if err != nil {
			return err
		}
	}

	log.Debug().Msgf("IPFS daemon is starting IPFS_PATH=%s", server.Repo)
	cmd := exec.Command("ipfs", "daemon")
	cmd.Env = []string{
		"IPFS_PATH=" + server.Repo,
		"IPFS_PROFILE=server",
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

	log.Debug().Msgf("IPFS daemon has started")

	testConnectionClient, err := ipfs_http.NewIPFSHttpClient(server.cancelContext.Ctx, server.ApiAddress())
	if err != nil {
		return err
	}

	ipfsReadyWaiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for ipfs server to be running: %s", server.ApiAddress()),
		MaxAttempts: 100,
		Delay:       time.Millisecond * 100,
		Logging:     false,
		Handler: func() (bool, error) {
			_, err := testConnectionClient.GetPeerId()
			if err != nil {
				return false, err
			}
			return true, nil
		},
	}

	err = ipfsReadyWaiter.Wait()
	if err != nil {
		return err
	}

	server.cancelContext.AddShutdownHandler(func() {
		err = system.RunCommand("kill", []string{
			"-9", fmt.Sprintf("%d", cmd.Process.Pid),
		})
		if err != nil {
			log.Error().Msgf("Error closing IPFS daemon %s", err.Error())
		} else {
			err := cmd.Wait()
			if err != nil {
				// we call wait so we don't have zombie processes
				// this is not a big deal as we are closing the process
			} else {
				log.Debug().Msgf("IPFS daemon has stopped")
			}
		}
	})

	return nil
}

func (server *IPFSDevServer) Address(port int) string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", port, server.Id)
}

func (server *IPFSDevServer) SwarmAddress() string {
	return server.Address(server.SwarmPort)
}

func (server *IPFSDevServer) ApiAddress() string {
	return server.Address(server.ApiPort)
}
