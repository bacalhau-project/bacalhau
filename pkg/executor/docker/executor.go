package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type DockerExecutor struct {
	// the global context for stopping any running jobs
	Ctx context.Context

	// used to allow multiple docker executors to run against the same docker server
	Id string

	// where do we copy the results from jobs temporarily?
	ResultsDir string

	// the storage providers we can implement for a job
	StorageProviders map[string]storage.StorageProvider

	Client *docker.DockerClient
}

func NewDockerExecutor(
	ctx context.Context,
	id string,
	storageProviders map[string]storage.StorageProvider,
) (*DockerExecutor, error) {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	dir, err := ioutil.TempDir("", "bacalhau-docker-executor")
	if err != nil {
		return nil, err
	}
	dockerExecutor := &DockerExecutor{
		Ctx:              ctx,
		Id:               id,
		ResultsDir:       dir,
		StorageProviders: storageProviders,
		Client:           dockerClient,
	}
	return dockerExecutor, nil
}

func (docker *DockerExecutor) getStorageProvider(engine string) (storage.StorageProvider, error) {
	return executor.GetStorageProvider(engine, docker.StorageProviders)
}

// check if docker itself is installed
func (docker *DockerExecutor) IsInstalled() (bool, error) {
	isRunning := docker.Client.IsInstalled()
	return isRunning, nil
}

func (docker *DockerExecutor) HasStorage(volume types.StorageSpec) (bool, error) {
	storage, err := docker.getStorageProvider(volume.Engine)
	if err != nil {
		return false, err
	}
	return storage.HasStorage(volume)
}

func (docker *DockerExecutor) RunJob(job *types.Job) (string, error) {

	spec := job.Spec

	if spec == nil {
		return "", fmt.Errorf("No job spec")

	}

	jobResultsDir, err := docker.ensureJobResultsDir(job)
	if err != nil {
		return "", err
	}

	// the actual mounts we will give to the container
	// these are paths for both input and output data
	mounts := []mount.Mount{}

	// loop over the job storage inputs and prepare them
	for _, inputStorage := range job.Spec.Inputs {
		storageProvider, err := docker.getStorageProvider(inputStorage.Engine)
		if err != nil {
			return "", err
		}
		volumeMount, err := storageProvider.PrepareStorage(inputStorage)
		if err != nil {
			return "", err
		}
		if volumeMount == nil {
			return "", fmt.Errorf("no volume mount was returned for input: %+v\n", inputStorage)
		}

		if volumeMount.Type == storage.STORAGE_VOLUME_TYPE_BIND {
			mounts = append(mounts, mount.Mount{
				Type: "bind",

				// this is an input volume so is read only
				ReadOnly: true,
				Source:   volumeMount.Source,
				Target:   volumeMount.Target,
			})
		} else {
			return "", fmt.Errorf("unknown storage volume type: %s\n", volumeMount.Type)
		}
	}

	// for this phase of the outputs we ignore the engine because it's just about collecting the
	// data from the job and keeping it locally
	// the engine property of the output storage spec is how we will "publish" the output volume
	// if and when the deal is settled
	for _, output := range job.Spec.Outputs {
		if output.Name == "" {
			return "", fmt.Errorf("output volume has no name: %+v\n", output)
		}

		if output.Path == "" {
			return "", fmt.Errorf("output volume has no path: %+v\n", output)
		}

		// create a mount so the output data does not need to be copied back to the host
		mounts = append(mounts, mount.Mount{
			Type: "bind",
			// this is an output volume so can be written to
			ReadOnly: false,
			// we create a named folder in the job results folder for this output
			Source: fmt.Sprintf("%s/%s", jobResultsDir, output.Name),
			// the path of the output volume is from the perspective of inside the container
			Target: output.Path,
		})
	}

	// let's pull the image down
	imagePullStream, err := docker.Client.Client.ImagePull(
		docker.Ctx,
		job.Spec.Vm.Image,
		dockertypes.ImagePullOptions{},
	)

	if system.IsDebug() {
		io.Copy(os.Stdout, imagePullStream)
	}

	if err = imagePullStream.Close(); err != nil {
		return "", err
	}

	docker.Client.Client.ContainerCreate(
		docker.Ctx,
		&container.Config{
			Image:      job.Spec.Vm.Image,
			Tty:        false,
			Env:        job.Spec.Vm.Env,
			Entrypoint: job.Spec.Vm.Entrypoint,
		},
		&container.HostConfig{
			Mounts: mounts,
		},
		&network.NetworkingConfig{},
		nil,
		docker.jobContainerName(job),
	)
	return jobResultsDir, nil
}

func cleanupDockerExecutor(ctx context.Context, executor *DockerExecutor) {
	<-ctx.Done()
	err := system.RunCommand("sudo", []string{
		"rm", "-rf",
		fmt.Sprintf("%s/*", executor.ResultsDir),
	})
	if err != nil {
		log.Error().Msgf("Docker executor stop error: %s", err.Error())
		return
	}
}

func (docker *DockerExecutor) jobContainerName(job *types.Job) string {
	return fmt.Sprintf("bacalhau-%s-%s", docker.Id, job.Id)
}

func (docker *DockerExecutor) jobResultsDir(job *types.Job) string {
	return fmt.Sprintf("%s/%s", docker.ResultsDir, job.Id)
}

func (docker *DockerExecutor) ensureJobResultsDir(job *types.Job) (string, error) {
	dir := docker.jobResultsDir(job)
	err := os.MkdirAll(dir, 0777)
	return dir, err
}
