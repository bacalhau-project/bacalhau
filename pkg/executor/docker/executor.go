package docker

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/resourceusage"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

const NanoCPUCoefficient = 1000000000

type Executor struct {
	// used to allow multiple docker executors to run against the same docker server
	ID string

	// where do we copy the results from jobs temporarily?
	ResultsDir string

	// the storage providers we can implement for a job
	StorageProviders map[string]storage.StorageProvider

	Client *dockerclient.Client
}

func NewExecutor(
	cm *system.CleanupManager,
	id string,
	storageProviders map[string]storage.StorageProvider,
) (*Executor, error) {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "bacalhau-docker-executor")
	if err != nil {
		return nil, err
	}

	de := &Executor{
		ID:               id,
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

func (e *Executor) getStorageProvider(ctx context.Context, engine string) (storage.StorageProvider, error) {
	return util.GetStorageProvider(ctx, engine, e.StorageProviders)
}

// IsInstalled checks if docker itself is installed.
func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return docker.IsInstalled(e.Client), nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()

	s, err := e.getStorageProvider(ctx, volume.Engine)
	if err != nil {
		return false, err
	}

	return s.HasStorageLocally(ctx, volume)
}

func (e *Executor) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	storageProvider, err := e.getStorageProvider(ctx, volume.Engine)
	if err != nil {
		return 0, err
	}
	return storageProvider.GetVolumeSize(ctx, volume)
}

// TODO: #289 Clean up RunJob
// nolint:funlen,gocyclo // will clean up
func (e *Executor) RunJob(ctx context.Context, j *executor.Job) (string, error) {
	ctx, span := newSpan(ctx, "RunJob")
	defer span.End()

	spec := j.Spec
	if spec == nil {
		return "", fmt.Errorf("no job spec provided to docker executor")
	}

	jobResultsDir, err := e.ensureJobResultsDir(j)
	if err != nil {
		return "", err
	}

	// the actual mounts we will give to the container
	// these are paths for both input and output data
	mounts := []mount.Mount{}

	// loop over the job storage inputs and prepare them
	for _, inputStorage := range j.Spec.Inputs {
		var storageProvider storage.StorageProvider
		storageProvider, err = e.getStorageProvider(ctx, inputStorage.Engine)
		if err != nil {
			return "", err
		}

		var volumeMount *storage.StorageVolume
		volumeMount, err = storageProvider.PrepareStorage(ctx, inputStorage)
		if err != nil {
			return "", err
		}

		if volumeMount == nil {
			return "", fmt.Errorf(
				"no volume mount was returned for input: %+v", inputStorage)
		}

		if volumeMount.Type == storage.StorageVolumeTypeBind {
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
	for _, output := range j.Spec.Outputs {
		if output.Name == "" {
			return "", fmt.Errorf("output volume has no name: %+v", output)
		}

		if output.Path == "" {
			return "", fmt.Errorf("output volume has no path: %+v", output)
		}

		srcd := fmt.Sprintf("%s/%s", jobResultsDir, output.Name)
		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
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
		// TODO: #283 work out why this does not work in github actions
		// err = docker.PullImage(e.Client, job.Spec.Vm.Image)

		stdout, err := system.RunCommandGetResults( // nolint:govet // shadowing ok
			"docker",
			[]string{"pull", j.Spec.Docker.Image},
		)
		if err != nil {
			return "", err
		}

		log.Trace().Msgf("Pull image output: %s\n%s", j.Spec.Docker.Image, stdout)
	}

	containerConfig := &container.Config{
		Image:           j.Spec.Docker.Image,
		Tty:             false,
		Env:             j.Spec.Docker.Env,
		Entrypoint:      j.Spec.Docker.Entrypoint,
		Labels:          e.jobContainerLabels(j),
		NetworkDisabled: true,
	}

	log.Trace().Msgf("Container: %+v %+v", containerConfig, mounts)

	resourceRequirements := resourceusage.ParseResourceUsageConfig(j.Spec.Resources)

	jobContainer, err := e.Client.ContainerCreate(
		ctx,
		containerConfig,
		&container.HostConfig{
			Mounts: mounts,
			Resources: container.Resources{
				Memory:   int64(resourceRequirements.Memory),
				NanoCPUs: int64(resourceRequirements.CPU * NanoCPUCoefficient),
			},
		},
		&network.NetworkingConfig{},
		nil,
		e.jobContainerName(j),
	)
	if err != nil {
		return "", err
	}

	defer e.cleanupJob(j)
	err = e.Client.ContainerStart(
		ctx,
		jobContainer.ID,
		dockertypes.ContainerStartOptions{},
	)
	if err != nil {
		return "", err
	}

	// TODO: we should record all logs and as much diagnostics as possible
	// in the error case so a user can debug why their job failed
	handleErrorLogs := func() {
		stdout, stderr, _ := docker.GetLogs(e.Client, jobContainer.ID)
		log.Error().Msgf("Container stdout: %s", stdout)
		log.Error().Msgf("Container stderr: %s", stderr)
	}

	statusCh, errCh := e.Client.ContainerWait(
		ctx,
		jobContainer.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err = <-errCh:
		if err != nil {
			handleErrorLogs()
			return "", err
		}
	case exitStatus := <-statusCh:
		if exitStatus.Error != nil {
			handleErrorLogs()
			return "", errors.New(exitStatus.Error.Message)
		}
		if exitStatus.StatusCode != 0 {
			handleErrorLogs()
			return "", fmt.Errorf("exit code was non zero: %d", exitStatus.StatusCode)
		}
	}

	log.Debug().Msgf("Container stopped: %s", jobContainer.ID)

	stdout, stderr, err := docker.GetLogs(e.Client, jobContainer.ID)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/stdout", jobResultsDir), []byte(stdout), util.OS_ALL_R|util.OS_USER_RW)
	if err != nil {
		msg := fmt.Sprintf("could not write results to stdout: %s", err)
		log.Error().Msg(msg)
		return "", errors.New(msg)
	}

	err = os.WriteFile(fmt.Sprintf("%s/stderr", jobResultsDir), []byte(stderr), util.OS_ALL_R|util.OS_USER_RW)
	if err != nil {
		msg := fmt.Sprintf("could not write results to stderr: %s", err)
		log.Error().Msg(msg)
		return "", errors.New(msg)
	}

	return jobResultsDir, nil
}

func (e *Executor) cleanupJob(job *executor.Job) {
	if config.ShouldKeepStack() {
		return
	}
	err := docker.RemoveContainer(e.Client, e.jobContainerName(job))
	if err != nil {
		log.Error().Msgf("Docker remove container error: %s", err.Error())
	}
}

func (e *Executor) cleanupAll() {
	if config.ShouldKeepStack() {
		return
	}
	containersWithLabel, err := docker.GetContainersWithLabel(e.Client, "bacalhau-executor", e.ID)
	if err != nil {
		log.Error().Msgf("Docker executor stop error: %s", err.Error())
		return
	}
	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	// nolint:gocritic // will fix when we care
	for _, container := range containersWithLabel {
		err = docker.RemoveContainer(e.Client, container.ID)
		if err != nil {
			log.Error().Msgf("Non-critical error cleaning up container: %s", err.Error())
		}
	}
}

func (e *Executor) jobContainerName(job *executor.Job) string {
	return fmt.Sprintf("bacalhau-%s-%s", e.ID, job.ID)
}

func (e *Executor) jobContainerLabels(job *executor.Job) map[string]string {
	return map[string]string{
		"bacalhau-executor": e.ID,
		"bacalhau-jobID":    job.ID,
	}
}

func (e *Executor) jobResultsDir(job *executor.Job) string {
	return fmt.Sprintf("%s/%s", e.ResultsDir, job.ID)
}

func (e *Executor) ensureJobResultsDir(job *executor.Job) (string, error) {
	dir := e.jobResultsDir(job)
	err := os.MkdirAll(dir, util.OS_ALL_RWX)
	info, _ := os.Stat(dir)
	log.Trace().Msgf("Created job results dir (%s). Permissions: %s", dir, info.Mode())
	return dir, err
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "executor/docker", apiName)
}

// Compile-time interface check:
var _ executor.Executor = (*Executor)(nil)
