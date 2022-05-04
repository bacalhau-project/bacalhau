package dockeripfs

import (
	"context"
	"fmt"
	"io/ioutil"
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
)

const BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE string = "binocarlos/bacalhau-ipfs-sidebar-image:v1"

type StorageDockerIPFS struct {
	Ctx context.Context
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	Id           string
	Isolated     bool
	IPFSClient   *ipfs_http.IPFSHttpClient
	DockerClient *docker.DockerClient
}

func NewStorageDockerIPFS(
	ctx context.Context,
	ipfsMultiAddress string,
	isolated bool,
) (*StorageDockerIPFS, error) {
	api, err := ipfs_http.NewIPFSHttpClient(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}
	peerId, err := api.GetPeerId()
	if err != nil {
		return nil, err
	}
	dockerClient, err := docker.NewDockerClient(ctx)
	if err != nil {
		return nil, err
	}
	StorageDockerIPFS := &StorageDockerIPFS{
		Ctx:          ctx,
		Id:           peerId,
		Isolated:     isolated,
		IPFSClient:   api,
		DockerClient: dockerClient,
	}
	return StorageDockerIPFS, nil
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

// _, err = system.RunCommandGetResults("docker", []string{
// 	"run",
// 	"-d",
// 	"--cap-add", "SYS_ADMIN",
// 	"--device", "/dev/fuse",
// 	"--name", BACALHAU_DOCKER_IPFS_SIDECAR_NAME,
// 	"--mount", fmt.Sprintf("type=bind,source=%s,target=/ipfs,bind-propagation=rshared", dir),
// 	"--privileged",
// 	"-e", fmt.Sprintf("BACALHAU_DISABLE_MDNS_DISCOVERY=%s", disableMdnsDiscoveryString),
// 	"-e", fmt.Sprintf("BACALHAU_DELETE_BOOTSTRAP_ADDRESSES=%s", deleteDefaultBootstrapAddressesString),
// 	"-e", fmt.Sprintf("BACALHAU_IPFS_PEER_ADDRESSES=%s", strings.Join(ipfsPeerAddresses, ",")),
// 	BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE,
// })
func (dockerIpfs *StorageDockerIPFS) PrepareStorage(volume types.StorageSpec) (*storage.PreparedStorageVolume, error) {
	dockerIpfs.Mutex.Lock()
	defer dockerIpfs.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := dockerIpfs.DockerClient.GetContainer(dockerIpfs.sidecarContainerName())
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
	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		return err
	}

	env := []string{}

	if dockerIpfs.Isolated {
		env = append(env, "BACALHAU_DISABLE_MDNS_DISCOVERY=true")
		env = append(env, "BACALHAU_DELETE_BOOTSTRAP_ADDRESSES=true")
	}

	addresses, err := dockerIpfs.IPFSClient.GetLocalAddrStrings()
	if err != nil {
		return err
	}

	env = append(env, fmt.Sprintf("BACALHAU_IPFS_PEER_ADDRESSES=%s", strings.Join(addresses, ",")))

	containerCreated, err := dockerIpfs.DockerClient.Client.ContainerCreate(
		dockerIpfs.Ctx,
		&container.Config{
			Image: BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE,
			Tty:   false,
			Env:   env,
		},
		&container.HostConfig{
			CapAdd: []string{
				"SYS_ADMIN",
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: dir,
					Target: "/ipfs",
					BindOptions: &mount.BindOptions{
						Propagation: mount.PropagationShared,
					},
				},
			},
			Privileged: true,
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

	err = dockerIpfs.DockerClient.Client.ContainerStart(dockerIpfs.Ctx, containerCreated.ID, dockertypes.ContainerStartOptions{})

	if err != nil {
		return err
	}

	return nil
}

func (dockerIpfs *StorageDockerIPFS) stopSidecar() error {
	sidecar, err := dockerIpfs.DockerClient.GetContainer(dockerIpfs.sidecarContainerName())
	if err != nil {
		return err
	}

	// some kind of sidecar container exists
	if sidecar == nil {
		return nil
	}

	return dockerIpfs.DockerClient.RemoveContainer(sidecar.ID)
}

func (dockerIpfs *StorageDockerIPFS) sidecarContainerName() string {
	return fmt.Sprintf("bacalhau-ipfs-%s", dockerIpfs.Id)
}
