package fusedocker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
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

func NewStorageProvider(ctx context.Context, cm *system.CleanupManager, ipfsAPIAddress string) (
	*StorageProvider, error) {
	api, err := ipfs.NewClient(ipfsAPIAddress)
	if err != nil {
		return nil, err
	}

	peerID, err := api.ID(ctx)
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
	storageHandler.Mutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "StorageHandler.Mutex",
	})
	storageHandler.Mutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "StorageHandler.Mutex",
	})

	cm.RegisterCallback(func() error {
		// TODO: #893 this shouldn't be reusing the context as there's the possibility that it's already canceled
		return cleanupStorageDriver(ctx, storageHandler)
	})

	log.Ctx(ctx).Debug().Msgf(
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

func (sp *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()

	return sp.IPFSClient.HasCID(ctx, volume.CID)
}

func (sp *StorageProvider) GetVolumeSize(ctx context.Context, _ model.StorageSpec) (uint64, error) {
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
	storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	_, span := newSpan(ctx, "PrepareStorage")
	defer span.End()

	err := sp.ensureSidecar(ctx, storageSpec.CID)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	cidMountPath, err := sp.getCidMountPath(ctx, storageSpec.CID)
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
	_ model.StorageSpec, _ storage.StorageVolume) error {
	_, span := newSpan(ctx, "CleanupStorage")
	defer span.End()

	return nil
}

func (sp *StorageProvider) Upload(context.Context, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (sp *StorageProvider) Explode(context.Context, model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (sp *StorageProvider) cleanSidecar(ctx context.Context) (*dockertypes.Container, error) {
	sidecar, err := sp.getSidecar(ctx)
	if err != nil {
		return nil, err
	}
	if sidecar != nil {
		// ahhh but it's not running
		// so let's remove what is there
		if sidecar.State != "running" {
			err = docker.RemoveContainer(ctx, sp.DockerClient, sidecar.ID)
			if err != nil {
				return nil, err
			}
			sidecar = nil
		}
	}
	return sidecar, nil
}

func (sp *StorageProvider) ensureSidecar(ctx context.Context, cid string) error {
	sp.Mutex.Lock()
	defer sp.Mutex.Unlock()

	// does the sidecar container already exist?
	// we lookup by name
	sidecar, err := sp.cleanSidecar(ctx)
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
				sidecar, err = sp.getSidecar(ctx)
				if err != nil {
					return false, err
				}

				if sidecar != nil {
					err = docker.RemoveContainer(ctx, sp.DockerClient, sidecar.ID)
					if err != nil {
						return false, err
					}
				}

				err = sp.startSidecar(ctx)
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
						return sp.canSeeFuseMount(ctx, cid), nil
					},
				}

				err = fileWaiter.Wait(ctx)
				if err != nil {
					return false, err
				}

				return true, nil
			},
		}

		err = sidecarWaiter.Wait(ctx)

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

	logs, err := docker.WaitForContainerLogs(ctx, sp.DockerClient, sidecarContainer.ID, MaxAttemptsForDocker, time.Second*2, "Daemon is ready")

	if err != nil {
		log.Ctx(ctx).Error().Msg(logs)
		stopErr := cleanupStorageDriver(ctx, sp)
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

func (sp *StorageProvider) getSidecar(ctx context.Context) (*dockertypes.Container, error) {
	return docker.GetContainer(ctx, sp.DockerClient, sp.sidecarContainerName())
}

// read from the running container what mount folder we have assigned
// we then use this for job container mounts for CID -> filepath volumes
func (sp *StorageProvider) getMountDir(ctx context.Context) (string, error) {
	sidecar, err := sp.getSidecar(ctx)
	if err != nil {
		return "", err
	}
	return getMountDirFromContainer(sidecar), nil
}

func (sp *StorageProvider) getCidMountPath(ctx context.Context, cid string) (string, error) {
	mountDir, err := sp.getMountDir(ctx)
	if err != nil {
		return "", err
	}
	if mountDir == "" {
		return "", fmt.Errorf("mount dir not found")
	}
	return fmt.Sprintf("%s/data/%s", mountDir, cid), nil
}

func (sp *StorageProvider) canSeeFuseMount(ctx context.Context, cid string) bool {
	testMountPath, err := sp.getCidMountPath(ctx, cid)
	if err != nil {
		return false
	}
	_, err = system.UnsafeForUserCodeRunCommand("sudo", []string{
		"timeout", "1s", "ls", "-la",
		testMountPath,
	})
	return err == nil
}

func cleanupStorageDriver(ctx context.Context, storageHandler *StorageProvider) error {
	// We have to use a separate context, rather than the one passed in to `NewExecutor`, as it may have already been
	// canceled and so would prevent us from performing any cleanup work.
	safeCtx := context.Background()

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
	}
	c, err := docker.GetContainer(safeCtx, dockerClient, storageHandler.sidecarContainerName())
	if err != nil {
		return fmt.Errorf("docker IPFS sidecar stop error: %s", err.Error())
	}
	if c != nil {
		err = docker.RemoveContainer(safeCtx, dockerClient, c.ID)
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
	log.Ctx(ctx).Debug().Msgf("Docker IPFS sidecar has stopped")
	return nil
}

func createMountDir() (string, error) {
	// create a temporary directory to mount the ipfs volume with fuse
	dir, err := os.MkdirTemp("", "bacalhau-ipfs")
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
	_, err := system.UnsafeForUserCodeRunCommand("sudo", []string{
		"umount",
		fmt.Sprintf("%s/data", mountDir),
	})
	if err != nil {
		return err
	}
	_, err = system.UnsafeForUserCodeRunCommand("sudo", []string{
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
var _ storage.Storage = (*StorageProvider)(nil)
