package fuse_docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

// a storage driver runs ipfs in a container
// that connects to an upstream IPFS node
// and fuse mounts to a bind with rshared mount propagation
// the result is a container that manages the host fuse mount
// we can then create docker volumes directly from
// host folders e.g. -v /tmp/ipfs_mnt/123:/file.txt

// TODO: this should come from a CI build
const BACALHAU_IPFS_FUSE_IMAGE string = "binocarlos/bacalhau-ipfs-sidecar-image:v1"
const BACALHAU_IPFS_FUSE_MOUNT = "/ipfs_mount"

type IpfsFuseDocker struct {
	cancelContext *system.CancelContext
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	Id           string
	IPFSClient   *ipfs_http.IPFSHttpClient
	DockerClient *dockerclient.Client
}

func NewIpfsFuseDocker(
	cancelContext *system.CancelContext,
	ipfsMultiAddress string,
) (*IpfsFuseDocker, error) {
	api, err := ipfs_http.NewIPFSHttpClient(cancelContext.Ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}
	peerId, err := api.GetPeerId()
	if err != nil {
		return nil, err
	}
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	storageHandler := &IpfsFuseDocker{
		cancelContext: cancelContext,
		Id:            peerId,
		IPFSClient:    api,
		DockerClient:  dockerClient,
	}

	cancelContext.AddShutdownHandler(func() {
		err := cleanupStorageDriver(storageHandler)
		if err != nil {
			log.Error().Msg(err.Error())
		}
	})

	log.Debug().Msgf("Docker IPFS storage initialized with address: %s", ipfsMultiAddress)

	return storageHandler, nil
}

func (dockerIpfs *IpfsFuseDocker) IsInstalled() (bool, error) {
	addresses, err := dockerIpfs.IPFSClient.GetLocalAddrs()
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("No multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (dockerIpfs *IpfsFuseDocker) HasStorage(volume types.StorageSpec) (bool, error) {
	return dockerIpfs.IPFSClient.HasCidLocally(volume.Cid)
}

// sometimes (for reasons we still need to work out) - the sidecar fuse mount container
// hangs on the initial "ls" - this is because of an IPFS network issue
// and restarting the container usually means it works next time
// so - let's put the "start sidecar" into a loop we try a few times
// TODO: work out what the underlying networking issue actually is
func (dockerIpfs *IpfsFuseDocker) PrepareStorage(storageSpec types.StorageSpec) (*types.StorageVolume, error) {
	err := dockerIpfs.ensureSidecar(storageSpec.Cid)
	if err != nil {
		return nil, err
	}

	cidMountPath, err := dockerIpfs.getCidMountPath(storageSpec.Cid)
	if err != nil {
		return nil, err
	}

	volume := &types.StorageVolume{
		Type:   storage.STORAGE_VOLUME_TYPE_BIND,
		Source: cidMountPath,
		Target: storageSpec.Path,
	}

	return volume, nil
}

// we don't need to cleanup individual storage because the fuse mount
// covers the whole of the ipfs namespace
func (dockerIpfs *IpfsFuseDocker) CleanupStorage(storageSpec types.StorageSpec, volume *types.StorageVolume) error {
	return nil
}

func (dockerIpfs *IpfsFuseDocker) cleanSidecar() (*dockertypes.Container, error) {
	sidecar, err := dockerIpfs.getSidecar()
	if err != nil {
		return nil, err
	}
	if sidecar != nil {
		// ahhh but it's not running
		// so let's remove what is there
		if sidecar.State != "running" {
			err = docker.RemoveContainer(dockerIpfs.DockerClient, sidecar.ID)
			if err != nil {
				return nil, err
			}
			sidecar = nil
		}
	}
	return sidecar, nil
}

func (dockerIpfs *IpfsFuseDocker) ensureSidecar(cid string) error {
	dockerIpfs.Mutex.Lock()
	defer dockerIpfs.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := dockerIpfs.cleanSidecar()
	if err != nil {
		return err
	}

	if sidecar == nil {

		// at this point - we know we are "starting" the container
		// and so we want to loop over:
		//  * start
		//  * wait for file
		//  * clean if error
		sidecarWaiter := &system.FunctionWaiter{
			Name:        "wait for ipfs fuse sidecar to start",
			MaxAttempts: 3,
			Delay:       time.Second * 1,
			Logging:     true,
			Handler: func() (bool, error) {

				sidecar, err := dockerIpfs.getSidecar()
				if err != nil {
					return false, err
				}

				if sidecar != nil {
					err = docker.RemoveContainer(dockerIpfs.DockerClient, sidecar.ID)
					if err != nil {
						return false, err
					}
				}

				err = dockerIpfs.startSidecar()
				if err != nil {
					return false, err
				}

				// this waits for the test fuse file to appear
				// as confirmation that our connection to the upstream
				fileWaiter := &system.FunctionWaiter{
					Name:        fmt.Sprintf("wait for ipfs fuse sidecar file to mount: %s", cid),
					MaxAttempts: 10,
					Delay:       time.Second * 1,
					Logging:     true,
					Handler: func() (bool, error) {
						return dockerIpfs.canSeeFuseMount(cid), nil
					},
				}

				err = fileWaiter.Wait()
				if err != nil {
					return false, err
				}

				return true, nil
			},
		}

		err = sidecarWaiter.Wait()

		if err != nil {
			return err
		}

	}

	return nil
}

func (dockerIpfs *IpfsFuseDocker) startSidecar() error {

	addresses, err := dockerIpfs.IPFSClient.GetSwarmAddresses()
	if err != nil {
		return err
	}

	mountDir, err := createMountDir()
	if err != nil {
		return err
	}

	gatewayPort, err := freeport.GetFreePort()
	if err != nil {
		return err
	}

	apiPort, err := freeport.GetFreePort()
	if err != nil {
		return err
	}

	swarmPort, err := freeport.GetFreePort()
	if err != nil {
		return err
	}

	sidecarContainer, err := dockerIpfs.DockerClient.ContainerCreate(
		dockerIpfs.cancelContext.Ctx,
		&container.Config{
			Image: BACALHAU_IPFS_FUSE_IMAGE,
			Tty:   false,
			Env: []string{
				// TODO: allow this to be configured - it's the bacalhau host ip
				// that will announce to the upstream ipfs server
				// if the upstream ipfs server is on the same host as bacalhau
				// then we can use 127.0.0.1 because the sidecar uses --net=host
				fmt.Sprintf("BACALHAU_IPFS_SWARM_ANNOUNCE_IP=%s", "127.0.0.1"),
				fmt.Sprintf("BACALHAU_IPFS_FUSE_MOUNT=%s", BACALHAU_IPFS_FUSE_MOUNT),
				fmt.Sprintf("BACALHAU_IPFS_PORT_GATEWAY=%d", gatewayPort),
				fmt.Sprintf("BACALHAU_IPFS_PORT_API=%d", apiPort),
				fmt.Sprintf("BACALHAU_IPFS_PORT_SWARM=%d", swarmPort),
				fmt.Sprintf("BACALHAU_IPFS_PEER_ADDRESSES=%s", strings.Join(addresses, ",")),
			},
		},
		&container.HostConfig{
			CapAdd: []string{
				"SYS_ADMIN",
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: mountDir,
					Target: BACALHAU_IPFS_FUSE_MOUNT,
					BindOptions: &mount.BindOptions{
						Propagation: mount.PropagationRShared,
					},
				},
			},
			NetworkMode: "host",
			Privileged:  true,
			Resources: container.Resources{
				Devices: []container.DeviceMapping{
					{
						PathOnHost:        "/dev/fuse",
						PathInContainer:   "/dev/fuse",
						CgroupPermissions: "rwm",
					},
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		dockerIpfs.sidecarContainerName(),
	)

	if err != nil {
		return err
	}

	err = dockerIpfs.DockerClient.ContainerStart(dockerIpfs.cancelContext.Ctx, sidecarContainer.ID, dockertypes.ContainerStartOptions{})

	if err != nil {
		return err
	}

	logs, err := docker.WaitForContainerLogs(dockerIpfs.DockerClient, sidecarContainer.ID, 5, time.Second*2, "Daemon is ready")

	if err != nil {
		log.Error().Msg(logs)
		stopErr := cleanupStorageDriver(dockerIpfs)
		if stopErr != nil {
			err = fmt.Errorf("Original Error: %s\nStop Error: %s\n", err.Error(), stopErr.Error())
		}
		return err
	}

	return nil
}

func (dockerIpfs *IpfsFuseDocker) sidecarContainerName() string {
	return fmt.Sprintf("bacalhau-ipfs-sidecar-%s", dockerIpfs.Id)
}

func (dockerIpfs *IpfsFuseDocker) getSidecar() (*dockertypes.Container, error) {
	return docker.GetContainer(dockerIpfs.DockerClient, dockerIpfs.sidecarContainerName())
}

// read from the running container what mount folder we have assigned
// we then use this for job container mounts for CID -> filepath volumes
func (dockerIpfs *IpfsFuseDocker) getMountDir() (string, error) {
	sidecar, err := dockerIpfs.getSidecar()
	if err != nil {
		return "", err
	}
	return getMountDirFromContainer(sidecar), nil
}

func (dockerIpfs *IpfsFuseDocker) getCidMountPath(cid string) (string, error) {
	mountDir, err := dockerIpfs.getMountDir()
	if err != nil {
		return "", err
	}
	if mountDir == "" {
		return "", fmt.Errorf("Mount dir not found")
	}
	return fmt.Sprintf("%s/data/%s", mountDir, cid), nil
}

func (dockerIpfs *IpfsFuseDocker) canSeeFuseMount(cid string) bool {
	testMountPath, err := dockerIpfs.getCidMountPath(cid)
	if err != nil {
		return false
	}
	_, err = system.RunCommandGetResults("sudo", []string{
		"timeout", "1s", "ls", "-la",
		testMountPath,
	})
	return err == nil
}

func cleanupStorageDriver(storageHandler *IpfsFuseDocker) error {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return fmt.Errorf("Docker IPFS sidecar stop error: %s", err.Error())
	}
	container, err := docker.GetContainer(dockerClient, storageHandler.sidecarContainerName())
	if err != nil {
		return fmt.Errorf("Docker IPFS sidecar stop error: %s", err.Error())
	}
	if container != nil {
		err = docker.RemoveContainer(dockerClient, container.ID)
		if err != nil {
			return fmt.Errorf("Docker IPFS sidecar stop error: %s", err.Error())
		}
	}
	mountDir := getMountDirFromContainer(container)
	if mountDir != "" {
		err = cleanupMountDir(mountDir)
		if err != nil {
			return fmt.Errorf("Docker IPFS sidecar stop error: %s", err.Error())
		}
	}
	log.Debug().Msgf("Docker IPFS sidecar has stopped")
	return nil
}

func createMountDir() (string, error) {
	// create a temporary directory to mount the ipfs volume with fuse
	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(fmt.Sprintf("%s/data", dir), 0777)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(fmt.Sprintf("%s/ipns", dir), 0777)
	if err != nil {
		return "", err
	}
	return dir, nil
}

func cleanupMountDir(mountDir string) error {
	err := system.RunCommand("sudo", []string{
		"umount",
		fmt.Sprintf("%s/data", mountDir),
	})
	if err != nil {
		return err
	}
	err = system.RunCommand("sudo", []string{
		"umount",
		fmt.Sprintf("%s/ipns", mountDir),
	})
	if err != nil {
		return err
	}
	return nil
}

func getMountDirFromContainer(container *dockertypes.Container) string {
	if container == nil {
		return ""
	}
	for _, mount := range container.Mounts {
		if mount.Destination == BACALHAU_IPFS_FUSE_MOUNT {
			return mount.Source
		}
	}
	return ""
}
