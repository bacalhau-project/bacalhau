package fuse_docker

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
	Ctx context.Context
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	Id           string
	IPFSClient   *ipfs_http.IPFSHttpClient
	DockerClient *dockerclient.Client
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

func (dockerIpfs *IpfsFuseDocker) PrepareStorage(storageSpec types.StorageSpec) (*types.StorageVolume, error) {

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

	volume := &types.StorageVolume{
		Type:   storage.STORAGE_VOLUME_TYPE_BIND,
		Source: fmt.Sprintf("%s/data/%s", mountdir, storageSpec.Cid),
		Target: storageSpec.Path,
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

func (dockerIpfs *IpfsFuseDocker) CleanupStorage(storageSpec types.StorageSpec, volume *types.StorageVolume) error {
	return nil
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
			err = docker.RemoveContainer(dockerIpfs.DockerClient, sidecar.ID)

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

	sidecarContainer, err := dockerIpfs.DockerClient.ContainerCreate(
		dockerIpfs.Ctx,
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

	err = dockerIpfs.DockerClient.ContainerStart(dockerIpfs.Ctx, sidecarContainer.ID, dockertypes.ContainerStartOptions{})

	if err != nil {
		return err
	}

	err = docker.WaitForContainerLogs(dockerIpfs.DockerClient, sidecarContainer.ID, 100, time.Millisecond*100, "Daemon is ready")

	if err != nil {
		return err
	}

	// TODO: we probably don't need this?
	time.Sleep(time.Second * 1)

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

func cleanupStorageDriver(ctx context.Context, storageHandler *IpfsFuseDocker) {
	<-ctx.Done()
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Error().Msgf("Docker IPFS sidecar stop error: %s", err.Error())
		return
	}
	container, err := docker.GetContainer(dockerClient, storageHandler.sidecarContainerName())
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

	err = docker.RemoveContainer(dockerClient, container.ID)
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
		if mount.Destination == BACALHAU_IPFS_FUSE_MOUNT {
			return mount.Source
		}
	}
	return ""
}
