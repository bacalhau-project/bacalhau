package ipfsdevstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/url"
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
	// For cleaning up IPFS daemons when shutting down:
	cm *system.CleanupManager

	ID          string
	Repo        string
	LogFile     string
	Isolated    bool
	CLI         *ipfs_cli.IPFSCli
	GatewayPort int
	APIPort     int
	SwarmPort   int
}

var OWNERONLYRW = 0600

func NewDevServer(cm *system.CleanupManager, isolated bool) (*IPFSDevServer, error) {
	repoDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary directory for ipfs repo: %s", err.Error())
	}

	logFile, err := ioutil.TempFile("", "bacalhau-ipfs-devstack")
	if err != nil {
		return nil, fmt.Errorf("could not create log file for ipfs repo: %s", err.Error())
	}

	gatewayPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("could not create random port for gateway: %s", err.Error())
	}

	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("could not create random port for api: %s", err.Error())
	}

	swarmPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("could not create random port for swarm: %s", err.Error())
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

	return &IPFSDevServer{
		cm:          cm,
		ID:          idResult.ID,
		Repo:        repoDir,
		LogFile:     logFile.Name(),
		CLI:         cli,
		Isolated:    isolated,
		GatewayPort: gatewayPort,
		APIPort:     apiPort,
		SwarmPort:   swarmPort,
	}, nil
}

// TODO: #286 Break this down - method is very large
func (server *IPFSDevServer) Start(connectToAddress string) error { // nolint:funlen,gocyclo // will fix
	if server.Isolated {
		_, err := server.CLI.Run([]string{
			"bootstrap", "rm", "--all",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"AutoNAT.ServiceMode",
			"disabled",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.EnableHolePunching",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.DisableNatPortMap",
			"--bool",
			"true",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.RelayClient.Enabled",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.RelayService.Enabled",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.Transports.Network.Relay",
			"--bool",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Swarm.Transports.Network.Relay",
			"--json",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Discovery.MDNS.Enabled",
			"--json",
			"false",
		})
		if err != nil {
			return err
		}

		_, err = server.CLI.Run([]string{
			"config",
			"Addresses.Announce",
			"--json",
			fmt.Sprintf(`["/ip4/127.0.0.1/tcp/%d"]`, server.SwarmPort),
		})
		if err != nil {
			return err
		}
	}

	_, err := server.CLI.Run([]string{
		"config",
		"Addresses.Gateway",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", server.GatewayPort),
	})
	if err != nil {
		return err
	}

	_, err = server.CLI.Run([]string{
		"config",
		"Addresses.API",
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", server.APIPort),
	})
	if err != nil {
		return err
	}

	_, err = server.CLI.Run([]string{
		"config",
		"Addresses.Swarm",
		"--json",
		fmt.Sprintf(`["/ip4/0.0.0.0/tcp/%d"]`, server.SwarmPort),
	})
	if err != nil {
		return err
	}

	if connectToAddress != "" {
		_, err = server.CLI.Run([]string{
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

	var logfile *os.File
	logfile, err = os.OpenFile(
		server.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, fs.FileMode(OWNERONLYRW))
	if err != nil {
		return err
	}

	cmd.Stderr = logfile
	cmd.Stdout = logfile

	err = cmd.Start()
	if err != nil {
		return err
	}
	log.Debug().Msgf("IPFS daemon has started")

	testConnectionClient, err := ipfs_http.NewIPFSHTTPClient(
		server.APIAddress())
	if err != nil {
		return err
	}

	ipfsReadyWaiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for ipfs server to be running: %s", server.APIAddress()),
		MaxAttempts: 100,
		Delay:       time.Millisecond * 100,
		Handler: func() (bool, error) {
			_, err = testConnectionClient.GetPeerID(context.Background())
			if err != nil {
				var expectedErr *url.Error
				if errors.As(err, &expectedErr) {
					return false, nil // connection not found, so we wait
				}

				return false, err // unexpected error
			}

			return true, nil
		},
	}

	err = ipfsReadyWaiter.Wait()
	if err != nil {
		return err
	}

	server.cm.RegisterCallback(func() error {
		err = system.RunCommand("kill", []string{
			"-9", fmt.Sprintf("%d", cmd.Process.Pid),
		})
		if err != nil {
			return err
		}

		if err := cmd.Wait(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				// we expect a non-zero exit code as we killed the process
			} else {
				return err
			}
		}

		log.Debug().Msgf("IPFS daemon has stopped.")
		return nil
	})

	return nil
}

func (server *IPFSDevServer) Address(port int) string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", port, server.ID)
}

func (server *IPFSDevServer) SwarmAddress() string {
	return server.Address(server.SwarmPort)
}

func (server *IPFSDevServer) APIAddress() string {
	return server.Address(server.APIPort)
}
