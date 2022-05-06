package ipfs_fuse_docker

import (
	"context"
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
	"github.com/filecoin-project/bacalhau/pkg/docker"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

const BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE string = "binocarlos/bacalhau-ipfs-sidebar-image:v1"
const BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_MOUNT = "/ipfs_mount"
const BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_SWARM_PORT = 4001

type IpfsFuseDocker struct {
	Ctx context.Context
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	Id           string
	IPFSClient   *ipfs_http.IPFSHttpClient
	DockerClient *docker.DockerClient
}

func NewIpfsFuseDocker(
	ctx context.Context,
	ipfsMultiAddress string,
) (*IpfsFuseDocker, error) {
	api, err := ipfs_http.NewIPFSHttpClient(ctx, ipfsMultiAddress)
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
		Ctx:          ctx,
		Id:           peerId,
		IPFSClient:   api,
		DockerClient: dockerClient,
	}

	// remove the sidecar when the context finishes
	go cleanupStorageDriver(ctx, storageHandler)

	log.Debug().Msgf("Docker IPFS sidecar Created")

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

/*
docker run --rm --name ipfs \
  --privileged \
  --cap-add SYS_ADMIN \
  --device /dev/fuse \
  --mount type=bind,source=/tmp/ipfs_test,target=/ipfs,bind-propagation=rshared \
  -e BACALHAU_DISABLE_MDNS_DISCOVERY=1 \
  -e BACALHAU_DELETE_BOOTSTRAP_ADDRESSES=1 \
  -e BACALHAU_IPFS_PEER_ADDRESSES=/ip4/192.168.1.151/tcp/4001/p2p/12D3KooWCrZmXHYaY4PYUP6GptpcwA4benxxcjfU9zyzkRZm6YYN \
  binocarlos/bacalhau-ipfs-sidebar-image:v1
*/
func (dockerIpfs *IpfsFuseDocker) PrepareStorage(storageSpec types.StorageSpec) (*storage.PreparedStorageVolume, error) {

	err := dockerIpfs.ensureSidecar()
	if err != nil {
		return nil, err
	}

	mountdir, err := dockerIpfs.getMountDir()
	if err != nil {
		return nil, err
	}
	if mountdir == "" {
		return nil, fmt.Errorf("Could not get mount dir")
	}

	volume := &storage.PreparedStorageVolume{
		Type:   "bind",
		Source: fmt.Sprintf("%s/data/%s", mountdir, storageSpec.Cid),
		Target: storageSpec.MountPath,
	}

	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for file to appear: %s", volume.Source),
		MaxAttempts: 100,
		Delay:       time.Millisecond * 100,
		Logging:     true,
		Handler: func() (bool, error) {
			_, err := system.RunCommandGetResults("sudo", []string{
				"timeout", "5s", "ls", "-la",
				volume.Source,
			})

			if err != nil {
				return false, err
			}
			return true, nil
		},
	}

	err = waiter.Wait()

	if err != nil {
		return nil, err
	}

	return volume, nil
}

func (dockerIpfs *IpfsFuseDocker) ensureSidecar() error {
	dockerIpfs.Mutex.Lock()
	defer dockerIpfs.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := dockerIpfs.getSidecar()
	if err != nil {
		return err
	}

	// some kind of sidecar container exists
	if sidecar != nil {

		// ahhh but it's not running
		// so let's remove what is there
		if sidecar.State != "running" {
			err = dockerIpfs.DockerClient.RemoveContainer(sidecar.ID)

			if err != nil {
				return err
			}

			sidecar = nil
		}
	}

	// do we need to start the sidecar container?
	if sidecar == nil {
		err = dockerIpfs.startSidecar()
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

	sidecarContainer, err := dockerIpfs.DockerClient.Client.ContainerCreate(
		dockerIpfs.Ctx,
		&container.Config{
			Image: BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE,
			Tty:   false,
			Env: []string{
				fmt.Sprintf("BACALHAU_IPFS_PORT_GATEWAY=%d", gatewayPort),
				fmt.Sprintf("BACALHAU_IPFS_PORT_API=%d", apiPort),
				fmt.Sprintf("BACALHAU_IPFS_PORT_SWARM=%d", swarmPort),
				"BACALHAU_DISABLE_MDNS_DISCOVERY=true",
				"BACALHAU_DELETE_BOOTSTRAP_ADDRESSES=true",
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
					Target: BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_MOUNT,
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

	err = dockerIpfs.DockerClient.Client.ContainerStart(dockerIpfs.Ctx, sidecarContainer.ID, dockertypes.ContainerStartOptions{})

	if err != nil {
		return err
	}

	err = system.WaitForContainerLogs(dockerIpfs.DockerClient, sidecarContainer.ID, 100, time.Millisecond*100, "Daemon is ready")

	if err != nil {
		return err
	}

	// TODO: we probably don't need this?
	time.Sleep(time.Second * 1)

	return nil
}

func (dockerIpfs *IpfsFuseDocker) stopSidecar() error {
	return dockerIpfs.DockerClient.RemoveContainer(dockerIpfs.sidecarContainerName())
}

func (dockerIpfs *IpfsFuseDocker) sidecarContainerName() string {
	return fmt.Sprintf("bacalhau-ipfs-sidecar-%s", dockerIpfs.Id)
}

func (dockerIpfs *IpfsFuseDocker) getSidecar() (*dockertypes.Container, error) {
	return dockerIpfs.DockerClient.GetContainer(dockerIpfs.sidecarContainerName())
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

func cleanupStorageDriver(ctx context.Context, storageHandler *IpfsFuseDocker) {
	<-ctx.Done()
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
		return
	}
	container, err := dockerClient.GetContainer(storageHandler.sidecarContainerName())
	if err != nil {
		log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
		return
	}
	if container == nil {
		return
	}
	mountDir := getMountDirFromContainer(container)

	if mountDir != "" {
		err = cleanupMountDir(mountDir)
		if err != nil {
			log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
			return
		}
	}

	err = dockerClient.RemoveContainer(container.ID)
	if err != nil {
		log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
		return
	}
	log.Debug().Msgf("Docker IPFS sidecar has stopped")
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
		if mount.Destination == BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_MOUNT {
			return mount.Source
		}
	}
	return ""
}
