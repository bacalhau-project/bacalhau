package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type DockerExecutor struct {
	// used to allow multiple docker executors to run against the same docker server
	Id string

	// where do we copy the results from jobs temporarily?
	ResultsDir string

	// the storage providers we can implement for a job
	StorageProviders map[string]storage.StorageProvider

	Client *dockerclient.Client
}

func NewDockerExecutor(
	cm *system.CleanupManager,
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

	de := &DockerExecutor{
		Id:               id,
		ResultsDir:       dir,
		StorageProviders: storageProviders,
		Client:           dockerClient,
	}

	cm.RegisterCallback(func() error {
		de.cleanupAll()
		return nil
	})

	return de, nil
}

func (dockerExecutor *DockerExecutor) getStorageProvider(engine string) (
	storage.StorageProvider, error) {

	return storage.GetStorageProvider(engine, dockerExecutor.StorageProviders)
}

// IsInstalled checks if docker itself is installed.
func (dockerExecutor *DockerExecutor) IsInstalled() (bool, error) {
	return docker.IsInstalled(dockerExecutor.Client), nil
}

func (dockerExecutor *DockerExecutor) HasStorage(volume types.StorageSpec) (
	bool, error) {

	storage, err := dockerExecutor.getStorageProvider(volume.Engine)
	if err != nil {
		return false, err
	}

	return storage.HasStorage(volume)
}

func (dockerExecutor *DockerExecutor) RunJob(job *types.Job) (string, error) {
	spec := job.Spec
	if spec == nil {
		return "", fmt.Errorf("no job spec provided to docker executor")
	}

	jobResultsDir, err := dockerExecutor.ensureJobResultsDir(job)
	if err != nil {
		return "", err
	}

	// the actual mounts we will give to the container
	// these are paths for both input and output data
	mounts := []mount.Mount{}

	// loop over the job storage inputs and prepare them
	for _, inputStorage := range job.Spec.Inputs {
		storageProvider, err := dockerExecutor.getStorageProvider(
			inputStorage.Engine)
		if err != nil {
			return "", err
		}

		volumeMount, err := storageProvider.PrepareStorage(inputStorage)
		if err != nil {
			return "", err
		}

		if volumeMount == nil {
			return "", fmt.Errorf(
				"no volume mount was returned for input: %+v", inputStorage)
		}

		if volumeMount.Type == storage.STORAGE_VOLUME_TYPE_BIND {
			log.Trace().Msgf("Input Volume: %+v %+v", inputStorage, volumeMount)

			mounts = append(mounts, mount.Mount{
				Type: "bind",

				// this is an input volume so is read only
				ReadOnly: true,
				Source:   volumeMount.Source,
				Target:   volumeMount.Target,
			})
		} else {
			return "", fmt.Errorf(
				"unknown storage volume type: %s", volumeMount.Type)
		}
	}

	// for this phase of the outputs we ignore the engine because it's just about collecting the
	// data from the job and keeping it locally
	// the engine property of the output storage spec is how we will "publish" the output volume
	// if and when the deal is settled
	for _, output := range job.Spec.Outputs {
		if output.Name == "" {
			return "", fmt.Errorf("output volume has no name: %+v", output)
		}

		if output.Path == "" {
			return "", fmt.Errorf("output volume has no path: %+v", output)
		}

		srcd := fmt.Sprintf("%s/%s", jobResultsDir, output.Name)
		err = os.Mkdir(srcd, 0755)
		if err != nil {
			return "", err
		}

		log.Trace().Msgf("Output Volume: %+v", output)

		// create a mount so the output data does not need to be copied back to the host
		mounts = append(mounts, mount.Mount{

			Type: "bind",
			// this is an output volume so can be written to
			ReadOnly: false,

			// we create a named folder in the job results folder for this output
			Source: srcd,

			// the path of the output volume is from the perspective of inside the container
			Target: output.Path,
		})
	}

	if os.Getenv("SKIP_IMAGE_PULL") == "" {
		// TODO: work out why this does not work in github actions
		// err = docker.PullImage(dockerExecutor.Client, job.Spec.Vm.Image)

		// if err != nil {
		// 	return "", err
		// }

		stdout, err := system.RunCommandGetResults(
			"docker",
			[]string{"pull", job.Spec.Vm.Image},
		)
		if err != nil {
			return "", err
		}

		log.Trace().Msgf("Pull image output: %s\n%s", job.Spec.Vm.Image, stdout)
	}

	containerConfig := &container.Config{
		Image:           job.Spec.Vm.Image,
		Tty:             false,
		Env:             job.Spec.Vm.Env,
		Entrypoint:      job.Spec.Vm.Entrypoint,
		Labels:          dockerExecutor.jobContainerLabels(job),
		NetworkDisabled: true,
	}

	log.Trace().Msgf("Container: %+v %+v", containerConfig, mounts)

	jobContainer, err := dockerExecutor.Client.ContainerCreate(
		context.TODO(),
		containerConfig,
		&container.HostConfig{
			Mounts: mounts,
		},
		&network.NetworkingConfig{},
		nil,
		dockerExecutor.jobContainerName(job),
	)
	if err != nil {
		return "", err
	}

	defer dockerExecutor.cleanupJob(job)
	err = dockerExecutor.Client.ContainerStart(
		context.TODO(),
		jobContainer.ID,
		dockertypes.ContainerStartOptions{},
	)
	if err != nil {
		return "", err
	}

	// TODO: we should record all logs and as much diagnostics as possible
	// in the error case so a user can debug why their job failed
	handleErrorLogs := func() {
		stdout, stderr, _ := docker.GetLogs(dockerExecutor.Client, jobContainer.ID)
		log.Error().Msgf("Container stdout: %s", stdout)
		log.Error().Msgf("Container stderr: %s", stderr)
	}

	statusCh, errCh := dockerExecutor.Client.ContainerWait(
		context.TODO(),
		jobContainer.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err := <-errCh:
		if err != nil {
			handleErrorLogs()
			return "", err
		}
	case exitStatus := <-statusCh:
		if exitStatus.Error != nil {
			handleErrorLogs()
			return "", fmt.Errorf(exitStatus.Error.Message)
		}
		if exitStatus.StatusCode != 0 {
			handleErrorLogs()
			return "", fmt.Errorf("exit code was non zero: %d", exitStatus.StatusCode)
		}
	}

	log.Debug().Msgf("Container stopped: %s", jobContainer.ID)

	stdout, stderr, err := docker.GetLogs(dockerExecutor.Client, jobContainer.ID)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/stdout", jobResultsDir), []byte(stdout), 0644)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/stderr", jobResultsDir), []byte(stderr), 0644)
	if err != nil {
		return "", err
	}

	return jobResultsDir, nil
}

func (dockerExecutor *DockerExecutor) cleanupJob(job *types.Job) {
	if system.ShouldKeepStack() {
		return
	}
	err := docker.RemoveContainer(dockerExecutor.Client, dockerExecutor.jobContainerName(job))
	if err != nil {
		log.Error().Msgf("Docker remove container error: %s", err.Error())
	}
}

func (dockerExecutor *DockerExecutor) cleanupAll() {
	if system.ShouldKeepStack() {
		return
	}
	containersWithLabel, err := docker.GetContainersWithLabel(dockerExecutor.Client, "bacalhau-executor", dockerExecutor.Id)
	if err != nil {
		log.Error().Msgf("Docker executor stop error: %s", err.Error())
		return
	}
	for _, container := range containersWithLabel {
		err = docker.RemoveContainer(dockerExecutor.Client, container.ID)
		if err != nil {
			log.Error().Msgf("Docker remove container error: %s", err.Error())
		}
	}
}

func (dockerExecutor *DockerExecutor) jobContainerName(job *types.Job) string {
	return fmt.Sprintf("bacalhau-%s-%s", dockerExecutor.Id, job.Id)
}

func (dockerExecutor *DockerExecutor) jobContainerLabels(job *types.Job) map[string]string {
	return map[string]string{
		"bacalhau-executor": dockerExecutor.Id,
	}
}

func (dockerExecutor *DockerExecutor) jobResultsDir(job *types.Job) string {
	return fmt.Sprintf("%s/%s", dockerExecutor.ResultsDir, job.Id)
}

func (dockerExecutor *DockerExecutor) ensureJobResultsDir(job *types.Job) (string, error) {
	dir := dockerExecutor.jobResultsDir(job)
	err := os.MkdirAll(dir, 0777)
	return dir, err
}
