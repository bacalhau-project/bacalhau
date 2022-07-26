package fusedocker

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
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// a storage driver runs ipfs in a container
// that connects to an upstream IPFS node
// and fuse mounts to a bind with rshared mount propagation
// the result is a container that manages the host fuse mount
// we can then create docker volumes directly from
// host folders e.g. -v /tmp/ipfs_mnt/123:/file.txt

// TODO: this should come from a CI build
const BacalhauIPFSFuseImage string = "binocarlos/bacalhau-ipfs-sidecar-image:v1"
const BacalhauIPFSFuseMount = "/ipfs_mount"
const MaxAttemptsForDocker = 5

type StorageProvider struct {
	// we have a single mutex per storage driver
	// (multuple of these might exist per docker server in the case of devstack)
	// the job of this mutex is to stop a race condition starting two sidecars
	Mutex        sync.Mutex
	ID           string
	IPFSClient   *ipfs.Client
	DockerClient *dockerclient.Client
}

func NewStorageProvider(cm *system.CleanupManager, ipfsAPIAddress string) (
	*StorageProvider, error) {
	api, err := ipfs.NewClient(ipfsAPIAddress)
	if err != nil {
		return nil, err
	}

	peerID, err := api.ID(context.Background())
	if err != nil {
		return nil, err
	}

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	storageHandler := &StorageProvider{
		ID:           peerID,
		IPFSClient:   api,
		DockerClient: dockerClient,
	}

	cm.RegisterCallback(func() error {
		return cleanupStorageDriver(storageHandler)
	})

	log.Debug().Msgf(
		"Docker IPFS storage initialized with address: %s", ipfsAPIAddress)
	return storageHandler, nil
}

func (sp *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	addresses, err := sp.IPFSClient.SwarmAddresses(ctx)
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf(
			"no multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()

	return sp.IPFSClient.HasCID(ctx, volume.Cid)
}

func (sp *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	_, span := newSpan(ctx, "GetVolumeResourceUsage")
	defer span.End()
	return 0, nil
}

// sometimes (for reasons we still need to work out) - the sidecar fuse mount container
// hangs on the initial "ls" - this is because of an IPFS network issue
// and restarting the container usually means it works next time
// so - let's put the "start sidecar" into a loop we try a few times
// TODO: work out what the underlying networking issue actually is
func (sp *StorageProvider) PrepareStorage(ctx context.Context,
	storageSpec storage.StorageSpec) (storage.StorageVolume, error) {
	_, span := newSpan(ctx, "PrepareStorage")
	defer span.End()

	err := sp.ensureSidecar(storageSpec.Cid)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	cidMountPath, err := sp.getCidMountPath(storageSpec.Cid)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: cidMountPath,
		Target: storageSpec.Path,
	}

	return volume, nil
}

// we don't need to cleanup individual storage because the fuse mount
// covers the whole of the ipfs namespace
func (sp *StorageProvider) CleanupStorage(ctx context.Context,
	storageSpec storage.StorageSpec, volume storage.StorageVolume) error {
	_, span := newSpan(ctx, "CleanupStorage")
	defer span.End()

	return nil
}

func (s *StorageProvider) Upload(ctx context.Context, localPath string) (storage.StorageSpec, error) {
	return storage.StorageSpec{}, fmt.Errorf("not implemented")
}

func (s *StorageProvider) Explode(ctx context.Context, spec storage.StorageSpec) ([]string, error) {
	return []string{}, fmt.Errorf("not implemented")
}

func (sp *StorageProvider) cleanSidecar() (*dockertypes.Container, error) {
	sidecar, err := sp.getSidecar()
	if err != nil {
		return nil, err
	}
	if sidecar != nil {
		// ahhh but it's not running
		// so let's remove what is there
		if sidecar.State != "running" {
			err = docker.RemoveContainer(sp.DockerClient, sidecar.ID)
			if err != nil {
				return nil, err
			}
			sidecar = nil
		}
	}
	return sidecar, nil
}

func (sp *StorageProvider) ensureSidecar(cid string) error {
	sp.Mutex.Lock()
	defer sp.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := sp.cleanSidecar()
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
			Handler: func() (bool, error) {
				sidecar, err = sp.getSidecar()
				if err != nil {
					return false, err
				}

				if sidecar != nil {
					err = docker.RemoveContainer(sp.DockerClient, sidecar.ID)
					if err != nil {
						return false, err
					}
				}

				err = sp.startSidecar(context.Background())
				if err != nil {
					return false, err
				}

				// this waits for the test fuse file to appear
				// as confirmation that our connection to the upstream
				fileWaiter := &system.FunctionWaiter{
					Name:        fmt.Sprintf("wait for ipfs fuse sidecar file to mount: %s", cid),
					MaxAttempts: 10,
					Delay:       time.Second * 1,
					Handler: func() (bool, error) {
						return sp.canSeeFuseMount(cid), nil
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

func (sp *StorageProvider) startSidecar(ctx context.Context) error {
	addresses, err := sp.IPFSClient.SwarmAddresses(ctx)
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

	sidecarContainer, err := sp.DockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: BacalhauIPFSFuseImage,
			Tty:   false,
			Env: []string{
				// TODO: allow this to be configured - it's the bacalhau host ip
				// that will announce to the upstream ipfs server
				// if the upstream ipfs server is on the same host as bacalhau
				// then we can use 127.0.0.1 because the sidecar uses --net=host
				fmt.Sprintf("BACALHAU_IPFS_SWARM_ANNOUNCE_IP=%s", "127.0.0.1"),
				fmt.Sprintf("BACALHAU_IPFS_FUSE_MOUNT=%s", BacalhauIPFSFuseMount),
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
					Target: BacalhauIPFSFuseMount,
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
		sp.sidecarContainerName(),
	)

	if err != nil {
		return err
	}

	err = sp.DockerClient.ContainerStart(ctx,
		sidecarContainer.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	logs, err := docker.WaitForContainerLogs(sp.DockerClient, sidecarContainer.ID, MaxAttemptsForDocker, time.Second*2, "Daemon is ready")

	if err != nil {
		log.Error().Msg(logs)
		stopErr := cleanupStorageDriver(sp)
		if stopErr != nil {
			err = fmt.Errorf("original error: %s\nstop error: %s", err.Error(), stopErr.Error())
		}
		return err
	}

	return nil
}

func (sp *StorageProvider) sidecarContainerName() string {
	return fmt.Sprintf("bacalhau-ipfs-sidecar-%s", sp.ID)
}

func (sp *StorageProvider) getSidecar() (*dockertypes.Container, error) {
	return docker.GetContainer(sp.DockerClient, sp.sidecarContainerName())
}

// read from the running container what mount folder we have assigned
// we then use this for job container mounts for CID -> filepath volumes
func (sp *StorageProvider) getMountDir() (string, error) {
	sidecar, err := sp.getSidecar()
	if err != nil {
		return "", err
	}
	return getMountDirFromContainer(sidecar), nil
}

func (sp *StorageProvider) getCidMountPath(cid string) (string, error) {
	mountDir, err := sp.getMountDir()
	if err != nil {
		return "", err
	}
	if mountDir == "" {
		return "", fmt.Errorf("mount dir not found")
	}
	return fmt.Sprintf("%s/data/%s", mountDir, cid), nil
}

func (sp *StorageProvider) canSeeFuseMount(cid string) bool {
	testMountPath, err := sp.getCidMountPath(cid)
	if err != nil {
		return false
	}
	_, err = system.RunCommandGetResults("sudo", []string{
		"timeout", "1s", "ls", "-la",
		testMountPath,
	})
	return err == nil
}

func cleanupStorageDriver(storageHandler *StorageProvider) error {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
	}
	c, err := docker.GetContainer(dockerClient, storageHandler.sidecarContainerName())
	if err != nil {
		return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
	}
	if c != nil {
		err = docker.RemoveContainer(dockerClient, c.ID)
		if err != nil {
			return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
		}
	}
	mountDir := getMountDirFromContainer(c)
	if mountDir != "" {
		err = cleanupMountDir(mountDir)
		if err != nil {
			return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
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
	err = os.MkdirAll(fmt.Sprintf("%s/data", dir), util.OS_ALL_RWX)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(fmt.Sprintf("%s/ipns", dir), util.OS_ALL_RWX)
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

func getMountDirFromContainer(c *dockertypes.Container) string {
	if c == nil {
		return ""
	}
	for _, mount := range c.Mounts {
		if mount.Destination == BacalhauIPFSFuseMount {
			return mount.Source
		}
	}
	return ""
}

func newSpan(ctx context.Context, apiName string) (
	context.Context, trace.Span) {
	return system.Span(ctx, "storage/ipfs/fusedocker", apiName)
}

// Compile-time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
