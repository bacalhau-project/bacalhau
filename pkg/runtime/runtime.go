package runtime

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

const IGNITE_IMAGE string = "binocarlos/bacalhau-ignite-image:v1"
const BACALHAU_LOGFILE = "/tmp/bacalhau.log"

const BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE string = "binocarlos/bacalhau-ipfs-sidebar-image:v1"
const BACALHAU_DOCKER_IPFS_SIDECAR_NAME string = "bacalhau-ipfs-sidecar"

type Runtime struct {
	Kind       string // "ignite" or "docker"
	doubleDash string
	Id         string
	Name       string
	Job        *types.JobSpec
	JobId      string
	stopChan   chan bool
}

func cleanEmpty(values []string) []string {
	result := []string{}
	for _, entry := range values {
		if entry != "" {
			result = append(result, entry)
		}
	}
	return result
}

func NewRuntime(job *types.Job) (*Runtime, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("bacalhau%s%s", job.Id, id.String())
	// allow CI to use docker in place of ignite
	kind := os.Getenv("BACALHAU_RUNTIME")
	doubleDash := ""
	if kind == "" {
		kind = "ignite"
		doubleDash = "--"
	}
	if !(kind == "ignite" || kind == "docker") {
		panic(fmt.Sprintf(
			`unsupported runtime requested via BACALHAU_RUNTIME (%s), `+
				`please specify one of "ignite" or "docker"`, kind,
		))
	}
	runtime := &Runtime{
		Kind:       kind,
		doubleDash: doubleDash,
		Id:         id.String(),
		JobId:      job.Id,
		Name:       name,
		Job:        job.Spec,
		stopChan:   make(chan bool),
	}
	return runtime, nil
}

func (runtime *Runtime) Stop() (string, error) {
	output, err := system.RunCommandGetResults("sudo", cleanEmpty([]string{
		runtime.Kind,
		"exec",
		runtime.Name,
		runtime.doubleDash, "ipfs", "swarm", "peers",
	}))
	threadLogger := logger.LoggerWithRuntimeInfo(runtime.JobId)
	if err != nil {
		threadLogger.Debug().Msgf("-----> ON CONTAINER SHUTDOWN, ipfs swarm peers errored: %s", err)
		// not returning err because this is just for debugging and we still want to clean up
	}
	if os.Getenv("KEEP_CONTAINERS") != "" {
		return "", nil
	}
	threadLogger.Debug().Msgf("-----> ON CONTAINER SHUTDOWN, ipfs swarm peers is: %s", output)
	runtime.stopChan <- true
	return system.RunCommandGetResults("sudo", []string{
		runtime.Kind,
		"rm",
		"-f",
		runtime.Name,
	})
}

var ipfsSidecarMutex sync.Mutex

// acquire a sidecar mutex to stop two of these functions running concurrently
// ask docker if our sidecar is already running
// if yes - return (later: maybe reconfigure it if the config changed)
// start our sidecar container
//   * environment for
//     * deleteDefaultBootstrapAddresses
//     * disableMdnsDiscovery
//     * ipfsPeerAddresses
//   * entrypoint.sh will action these
//   * run test for fuse to be up and running
func (runtime *Runtime) EnsureIpfsSidecarRunning(
	// are we connecting to the public IPFS network or a private devstack?
	deleteDefaultBootstrapAddresses bool,
	disableMdnsDiscovery bool,
	ipfsPeerAddresses []string,
) error {

	ipfsSidecarMutex.Lock()
	defer ipfsSidecarMutex.Unlock()

	result, err := system.RunCommandGetResults("docker", []string{
		"ps", "-q", "--filter",
		fmt.Sprintf("name=%s,status=running", BACALHAU_DOCKER_IPFS_SIDECAR_NAME),
	})

	if err != nil {
		return err
	}

	// docker ps will return an empty string if there's no container matching
	// the name
	if result != "" {
		return nil
	}

	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		return err
	}
	err = os.MkdirAll(fmt.Sprintf("%s/data", dir), 0777)
	if err != nil {
		return err
	}
	err = os.MkdirAll(fmt.Sprintf("%s/ipns", dir), 0777)
	if err != nil {
		return err
	}

	// turn bools into env
	deleteDefaultBootstrapAddressesString := ""
	disableMdnsDiscoveryString := ""

	if deleteDefaultBootstrapAddresses {
		deleteDefaultBootstrapAddressesString = "true"
	}

	if disableMdnsDiscovery {
		disableMdnsDiscoveryString = "true"
	}

	_, err = system.RunCommandGetResults("docker", []string{
		"run",
		"-d",
		"--cap-add", "SYS_ADMIN",
		"--device", "/dev/fuse",
		"--name", BACALHAU_DOCKER_IPFS_SIDECAR_NAME,
		"--mount", fmt.Sprintf("type=bind,source=%s,target=/ipfs,bind-propagation=rshared", dir),
		"--privileged",
		"-e", fmt.Sprintf("BACALHAU_DISABLE_MDNS_DISCOVERY=%s", disableMdnsDiscoveryString),
		"-e", fmt.Sprintf("BACALHAU_DELETE_BOOTSTRAP_ADDRESSES=%s", deleteDefaultBootstrapAddressesString),
		"-e", fmt.Sprintf("BACALHAU_IPFS_PEER_ADDRESSES=%s", strings.Join(ipfsPeerAddresses, ",")),
		BACALHAU_DOCKER_IPFS_SIDECAR_IMAGE,
	})

	if err != nil {
		return err
	}

	// TODO: do an actual test that ipfs mount is up and running
	time.Sleep(time.Second * 5)

	return nil
}

// TODO: mount input data files
// TODO: mount output data files
// psrecord invoke the job that we have prepared at /job.sh
// copy the psrecord metrics out of the runtime
// TODO: bunlde the results data and metrics
// docker run --rm -ti --name ipfs_client  --mount type=bind,source=/ipfs/data,target=/ipfs,bind-propagation=rshared ubuntu bash
func (runtime *Runtime) RunJob(resultsFolder string) error {
	threadLogger := logger.LoggerWithRuntimeInfo(runtime.JobId)

	log.Info().Msgf(`
========================================
Running job: %s %s
========================================
`, runtime.Job.Image, runtime.Job.Entrypoint)

	err, stdout, stderr := system.RunTeeCommand("docker", cleanEmpty([]string{
		"run",
		"--rm",
		"--name",

		runtime.Kind,
		"exec",
		runtime.Name,
		runtime.doubleDash, "psrecord", "bash /job.sh", "--log", "/tmp/metrics.log", "--plot", "/tmp/metrics.png", "--include-children",
	}))

	if err != nil {
		return err
	}

	// write the command stdout & stderr to the results dir
	threadLogger.Debug().Msgf("Writing stdout to %s/stdout.log", resultsFolder)

	directoryExists := false
	if _, err := os.Stat(resultsFolder); !errors.Is(err, exec.ErrNotFound) {
		threadLogger.Debug().Msgf("Directory found: %s", resultsFolder)
		directoryExists = true
	} else {
		threadLogger.Debug().Msgf("Directory NOT found: %s", resultsFolder)
	}

	threadLogger.Debug().Msgf("Expected folder %s exists?: %t", resultsFolder, directoryExists)

	err = os.WriteFile(fmt.Sprintf("%s/stdout.log", resultsFolder), stdout.Bytes(), 0644)
	if err != nil {
		return err
	}

	threadLogger.Debug().Msgf("Writing stderr to %s/stderr.log\n", resultsFolder)
	err = os.WriteFile(fmt.Sprintf("%s/stderr.log", resultsFolder), stderr.Bytes(), 0644)
	if err != nil {
		return err
	}

	threadLogger.Info().Msgf("Finished writing results of job (Id: %s) to results folder (%s).", runtime.Id, resultsFolder)

	// copy the psrecord metrics out of the runtime
	filesToCopy := []string{
		"metrics.log",
		"metrics.png",
	}

	for _, file := range filesToCopy {
		threadLogger.Debug().Msgf("Copying files - Writing %s to %s/%s\n", file, resultsFolder, file)
		err = system.RunCommand("sudo", []string{
			runtime.Kind,
			"cp",
			fmt.Sprintf("%s:/tmp/%s", runtime.Name, file),
			fmt.Sprintf("%s/%s", resultsFolder, file),
		})
	}

	return err
}
