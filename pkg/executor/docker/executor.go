package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

const NanoCPUCoefficient = 1000000000

type Executor struct {
	// used to allow multiple docker executors to run against the same docker server
	ID string

	// the storage providers we can implement for a job
	StorageProvider storage.StorageProvider

	Client *dockerclient.Client
}

func NewExecutor(
	ctx context.Context,
	cm *system.CleanupManager,
	id string,
	storageProvider storage.StorageProvider,
) (*Executor, error) {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	de := &Executor{
		ID:              id,
		StorageProvider: storageProvider,
		Client:          dockerClient,
	}

	cm.RegisterCallback(func() error {
		de.cleanupAll(ctx)
		return nil
	})

	return de, nil
}

func (e *Executor) getStorage(ctx context.Context, engine model.StorageSourceType) (storage.Storage, error) {
	return e.StorageProvider.GetStorage(ctx, engine)
}

// IsInstalled checks if docker itself is installed.
func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return docker.IsInstalled(ctx, e.Client), nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/executor/docker/Executor.HasStorageLocally")
	defer span.End()

	s, err := e.getStorage(ctx, volume.StorageSource)
	if err != nil {
		return false, err
	}

	return s.HasStorageLocally(ctx, volume)
}

func (e *Executor) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	storageProvider, err := e.getStorage(ctx, volume.StorageSource)
	if err != nil {
		return 0, err
	}
	return storageProvider.GetVolumeSize(ctx, volume)
}

//nolint:funlen,gocyclo // will clean up
func (e *Executor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/executor/docker.RunShard")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)
	system.AddNodeIDFromBaggageToSpan(ctx, span)

	shardStorageSpec, err := jobutils.GetShardStorageSpec(ctx, shard, e.StorageProvider)
	if err != nil {
		return &model.RunCommandResult{}, err
	}

	inputStorageSpecs := []model.StorageSpec{}
	inputStorageSpecs = append(inputStorageSpecs, shard.Job.Spec.Contexts...)
	inputStorageSpecs = append(inputStorageSpecs, shardStorageSpec...)

	inputVolumes, err := storage.ParallelPrepareStorage(ctx, e.StorageProvider, inputStorageSpecs)
	if err != nil {
		return &model.RunCommandResult{}, err
	}

	// the actual mounts we will give to the container
	// these are paths for both input and output data
	mounts := []mount.Mount{}
	for spec, volumeMount := range inputVolumes {
		if volumeMount.Type == storage.StorageVolumeConnectorBind {
			log.Ctx(ctx).Trace().Msgf("Input Volume: %+v %+v", spec, volumeMount)
			mounts = append(mounts, mount.Mount{
				Type: mount.TypeBind,
				// this is an input volume so is read only
				ReadOnly: true,
				Source:   volumeMount.Source,
				Target:   volumeMount.Target,
			})
		} else {
			return &model.RunCommandResult{}, fmt.Errorf("unknown storage volume type: %s", volumeMount.Type)
		}
	}

	// for this phase of the outputs we ignore the engine because it's just about collecting the
	// data from the job and keeping it locally
	// the engine property of the output storage spec is how we will "publish" the output volume
	// if and when the deal is settled
	for _, output := range shard.Job.Spec.Outputs {
		if output.Name == "" {
			err = fmt.Errorf("output volume has no name: %+v", output)
			return &model.RunCommandResult{ErrorMsg: err.Error()}, err
		}

		if output.Path == "" {
			err = fmt.Errorf("output volume has no path: %+v", output)
			return &model.RunCommandResult{ErrorMsg: err.Error()}, err
		}

		srcd := filepath.Join(jobResultsDir, output.Name)
		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return &model.RunCommandResult{ErrorMsg: err.Error()}, err
		}

		log.Ctx(ctx).Trace().Msgf("Output Volume: %+v", output)

		// create a mount so the output data does not need to be copied back to the host
		mounts = append(mounts, mount.Mount{

			Type: mount.TypeBind,
			// this is an output volume so can be written to
			ReadOnly: false,

			// we create a named folder in the job results folder for this output
			Source: srcd,

			// the path of the output volume is from the perspective of inside the container
			Target: output.Path,
		})
	}

	if os.Getenv("SKIP_IMAGE_PULL") == "" {
		if err := docker.PullImage(ctx, e.Client, shard.Job.Spec.Docker.Image); err != nil { //nolint:govet // ignore err shadowing
			//nolint:stylecheck // Error message for user
			err = fmt.Errorf(`Could not pull image - could be due to repo/image not existing,
 or registry needing authorization. %s: %s`, shard.Job.Spec.Docker.Image, err)
			return returnStdErrWithErr(err.Error(), err), err
		}
	}

	// json the job spec and pass it into all containers
	// TODO: check if this will overwrite a user supplied version of this value
	// (which is what we actually want to happen)
	log.Ctx(ctx).Debug().Msgf("Job Spec: %+v", shard.Job.Spec)
	jsonJobSpec, err := model.JSONMarshalWithMax(shard.Job.Spec)
	if err != nil {
		return &model.RunCommandResult{ErrorMsg: err.Error()}, err
	}
	log.Ctx(ctx).Debug().Msgf("Job Spec JSON: %s", jsonJobSpec)

	useEnv := append(shard.Job.Spec.Docker.EnvironmentVariables, fmt.Sprintf("BACALHAU_JOB_SPEC=%s", string(jsonJobSpec))) //nolint:gocritic

	containerConfig := &container.Config{
		Image:           shard.Job.Spec.Docker.Image,
		Tty:             false,
		Env:             useEnv,
		Entrypoint:      shard.Job.Spec.Docker.Entrypoint,
		Labels:          e.jobContainerLabels(shard.Job),
		NetworkDisabled: true,
		WorkingDir:      shard.Job.Spec.Docker.WorkingDirectory,
	}

	log.Ctx(ctx).Trace().Msgf("Container: %+v %+v", containerConfig, mounts)

	resourceRequirements := capacitymanager.ParseResourceUsageConfig(shard.Job.Spec.Resources)

	// Create GPU request if the job requests it
	var deviceRequests []container.DeviceRequest
	if resourceRequirements.GPU > 0 {
		deviceRequests = append(deviceRequests,
			container.DeviceRequest{
				DeviceIDs:    []string{"0"}, // TODO: how do we know which device ID to use?
				Capabilities: [][]string{{"gpu"}},
			},
		)
		log.Ctx(ctx).Trace().Msgf("Adding %d GPUs to request", resourceRequirements.GPU)
	}

	jobContainer, err := e.Client.ContainerCreate(
		ctx,
		containerConfig,
		&container.HostConfig{
			Mounts: mounts,
			Resources: container.Resources{
				Memory:         int64(resourceRequirements.Memory),
				NanoCPUs:       int64(resourceRequirements.CPU * NanoCPUCoefficient),
				DeviceRequests: deviceRequests,
			},
		},
		&network.NetworkingConfig{},
		nil,
		e.jobContainerName(shard),
	)
	if err != nil {
		return returnStdErrWithErr("failed to create container: ", err), err
	}

	containerStartError := e.Client.ContainerStart(
		ctx,
		jobContainer.ID,
		dockertypes.ContainerStartOptions{},
	)
	if containerStartError != nil {
		// Special error to alert people about bad executable
		internalContainerStartErrorMsg := "failed to start container: "
		if strings.Contains(containerStartError.Error(), "executable file not found") {
			internalContainerStartErrorMsg = "Executable file not found: " + containerStartError.Error()
		}
		internalContainerStartError := fmt.Errorf(internalContainerStartErrorMsg)
		return returnStdErrWithErr(internalContainerStartError.Error(),
				internalContainerStartError),
			internalContainerStartError
	}

	defer e.cleanupJob(ctx, shard)

	// the idea here is even if the container errors
	// we want to capture stdout, stderr and feed it back to the user
	var containerError error
	var containerExitStatusCode int64
	statusCh, errCh := e.Client.ContainerWait(
		ctx,
		jobContainer.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err = <-errCh:
		containerError = err
	case exitStatus := <-statusCh:
		containerExitStatusCode = exitStatus.StatusCode
		if exitStatus.Error != nil {
			containerError = errors.New(exitStatus.Error.Message)
		}
	}

	log.Ctx(ctx).Debug().Msgf("Capturing stdout/stderr for container %s", jobContainer.ID)
	stdoutFilename := fmt.Sprintf("%s/stdout", jobResultsDir)
	stderrFilename := fmt.Sprintf("%s/stderr", jobResultsDir)

	log.Ctx(ctx).Debug().Msgf("Capturing stdout to %s", stdoutFilename)
	log.Ctx(ctx).Debug().Msgf("Capturing stderr to %s", stderrFilename)

	runResult, err := system.RunCommandResultsToDisk(
		"docker",
		[]string{
			"logs",
			"-f",
			jobContainer.ID,
		},
		stdoutFilename,
		stderrFilename,
	)
	if err != nil {
		err = fmt.Errorf("failed to get logs: %w", err)
		return &model.RunCommandResult{ErrorMsg: err.Error()}, err
	}

	runResult.ExitCode = int(containerExitStatusCode)
	if containerError != nil {
		runResult.ErrorMsg = containerError.Error()
	}
	if runResult.ExitCode != 0 {
		if runResult.ErrorMsg == "" {
			runResult.ErrorMsg = fmt.Sprintf("exit code was not zero: %d", containerExitStatusCode)
		}
		log.Ctx(ctx).Info().Msgf("container error %s", runResult.ErrorMsg)
	}

	log.Ctx(ctx).Trace().Msgf("Writing exit code for container %s", jobContainer.ID)
	err = os.WriteFile(
		fmt.Sprintf("%s/exitCode", jobResultsDir),
		[]byte(fmt.Sprintf("%d", containerExitStatusCode)),
		util.OS_ALL_R|util.OS_USER_RW,
	)
	if err != nil {
		runResult.ErrorMsg = errors.Wrap(err, "could not write results to exitCode: ").Error()
		log.Ctx(ctx).Error().Msg(runResult.ErrorMsg)
		return runResult, err
	}
	log.Ctx(ctx).Debug().Msgf("Wrote exit code %d to %s/exitCode", containerExitStatusCode, jobResultsDir)
	log.Ctx(ctx).Debug().Msgf("Returning RunOutput %+v", runResult)

	return runResult, err
}

func (e *Executor) CancelShard(ctx context.Context, shard model.JobShard) error {
	return docker.StopContainer(ctx, e.Client, e.jobContainerName(shard))
}

func returnStdErrWithErr(msg string, err error) *model.RunCommandResult {
	log.Debug().Msgf("Returning error %s", msg)
	log.Debug().Msgf("Returning error %s", err.Error())
	return &model.RunCommandResult{
		STDERR:   err.Error(),
		ErrorMsg: errors.Wrap(err, msg).Error(),
	}
}
func (e *Executor) cleanupJob(ctx context.Context, shard model.JobShard) {
	if config.ShouldKeepStack() {
		return
	}

	err := docker.RemoveContainer(ctx, e.Client, e.jobContainerName(shard))
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Docker remove container error: %s", err.Error())
		debug.PrintStack()
	}
}

func (e *Executor) cleanupAll(ctx context.Context) {
	if config.ShouldKeepStack() {
		return
	}

	// We have to use a separate context, rather than the one passed in to `NewExecutor`, as it may have already been
	// canceled and so would prevent us from performing any cleanup work.
	safeCtx := context.Background()

	log.Ctx(ctx).Debug().Msgf("Cleaning up all bacalhau containers for executor %s...", e.ID)
	containersWithLabel, err := docker.GetContainersWithLabel(safeCtx, e.Client, "bacalhau-executor", e.ID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Docker executor stop error: %s", err.Error())
		return
	}
	for _, container := range containersWithLabel {
		if err := docker.RemoveContainer(safeCtx, e.Client, container.ID); err != nil { //nolint:govet // ignore err shadowing
			log.Ctx(ctx).Err(err).Msgf("Non-critical error cleaning up container")
		}
	}
}

func (e *Executor) jobContainerName(shard model.JobShard) string {
	return fmt.Sprintf("bacalhau-%s-%s-%d", e.ID, shard.Job.ID, shard.Index)
}

func (e *Executor) jobContainerLabels(job *model.Job) map[string]string {
	return map[string]string{
		"bacalhau-executor": e.ID,
		"bacalhau-jobID":    job.ID,
	}
}

// Compile-time interface check:
var _ executor.Executor = (*Executor)(nil)
