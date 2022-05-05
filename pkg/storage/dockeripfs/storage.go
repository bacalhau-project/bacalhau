package dockeripfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

const BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE string = "binocarlos/bacalhau-ipfs-sidebar-image:v1"
const BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_MOUNT = "/ipfs_mount"
const BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_SWARM_PORT = 4001

type StorageDockerIPFS struct {
	Ctx context.Context
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	Id           string
	IPFSClient   *ipfs_http.IPFSHttpClient
	DockerClient *docker.DockerClient
}

func NewStorageDockerIPFS(
	ctx context.Context,
	ipfsMultiAddress string,
) (*StorageDockerIPFS, error) {
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
	storageHandler := &StorageDockerIPFS{
		Ctx:          ctx,
		Id:           peerId,
		IPFSClient:   api,
		DockerClient: dockerClient,
	}

	// remove the sidecar when the context finishes
	go func(ctx context.Context, storage *StorageDockerIPFS) {
		<-ctx.Done()
		dockerClient, err := docker.NewDockerClient()
		if err != nil {
			log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
			return
		}
		dockerClient.RemoveContainer(storage.sidecarContainerName())
		log.Debug().Msgf("Docker IPFS sidecar has stopped")
	}(ctx, storageHandler)

	log.Debug().Msgf("Docker IPFS sidecar Created")

	return storageHandler, nil
}

func (dockerIpfs *StorageDockerIPFS) IsInstalled() (bool, error) {
	addresses, err := dockerIpfs.IPFSClient.GetLocalAddrs()
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("No multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (dockerIpfs *StorageDockerIPFS) HasStorage(volume types.StorageSpec) (bool, error) {
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
func (dockerIpfs *StorageDockerIPFS) PrepareStorage(volume types.StorageSpec) (*storage.PreparedStorageVolume, error) {

	dockerIpfs.Mutex.Lock()
	defer dockerIpfs.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := dockerIpfs.getSidecar()
	if err != nil {
		return nil, err
	}

	// some kind of sidecar container exists
	if sidecar != nil {

		// ahhh but it's not running
		// so let's remove what is there
		if sidecar.State != "running" {
			err = dockerIpfs.DockerClient.RemoveContainer(sidecar.ID)

			if err != nil {
				return nil, err
			}

			sidecar = nil
		}
	}

	// do we need to start the sidecar container?
	if sidecar == nil {
		err = dockerIpfs.startSidecar()
		if err != nil {
			return nil, err
		}
	}

	//dockerIpfs.DockerClient.Client.
	return &storage.PreparedStorageVolume{
		Type:   "bind",
		Source: "apples",
		Target: "pears",
	}, nil
}

func (dockerIpfs *StorageDockerIPFS) startSidecar() error {

	addresses, err := dockerIpfs.IPFSClient.GetSwarmAddresses()
	if err != nil {
		return err
	}

	mountDir, err := dockerIpfs.createMountDir()
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
						Propagation: mount.PropagationShared,
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

	return nil
}

func (dockerIpfs *StorageDockerIPFS) stopSidecar() error {
	return dockerIpfs.DockerClient.RemoveContainer(dockerIpfs.sidecarContainerName())
}

func (dockerIpfs *StorageDockerIPFS) sidecarContainerName() string {
	return fmt.Sprintf("bacalhau-ipfs-sidecar-%s", dockerIpfs.Id)
}

func (dockerIpfs *StorageDockerIPFS) createMountDir() (string, error) {
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

func (dockerIpfs *StorageDockerIPFS) getSidecar() (*dockertypes.Container, error) {
	return dockerIpfs.DockerClient.GetContainer(dockerIpfs.sidecarContainerName())
}

// read from the running container what mount folder we have assigned
// we then use this for job container mounts for CID -> filepath volumes
func (dockerIpfs *StorageDockerIPFS) getMountDir() (string, error) {
	sidecar, err := dockerIpfs.getSidecar()
	if err != nil {
		return "", err
	}
	if sidecar == nil {
		return "", nil
	}
	for _, mount := range sidecar.Mounts {
		if mount.Destination == BACALHAU_DOCKER_IPFS_SIDECAR_INTERNAL_MOUNT {
			return mount.Source, nil
		}
	}
	return "", nil
}
