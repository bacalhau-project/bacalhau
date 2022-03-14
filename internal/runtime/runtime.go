package runtime

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

const IGNITE_IMAGE string = "binocarlos/bacalhau-ignite-image:v1"
const BACALHAU_LOGFILE = "/tmp/bacalhau.log"

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

// start the runtime so we can exec to prepare and run the job
func (runtime *Runtime) Start() (string, error) {
	if runtime.Kind == "ignite" {
		return system.RunCommandGetResults("sudo", []string{
			"ignite",
			"run",
			IGNITE_IMAGE,
			"--name",
			runtime.Name,
			"--cpus",
			fmt.Sprintf("%d", runtime.Job.Cpu),
			"--memory",
			fmt.Sprintf("%dGB", runtime.Job.Memory),
			"--size",
			fmt.Sprintf("%dGB", runtime.Job.Disk),
			"--ssh",
		})
	} else {
		return system.RunCommandGetResults("sudo", []string{
			"docker",
			"run",
			"--privileged",
			"-d",
			"--rm",
			"--name",
			runtime.Name,
			"--entrypoint",
			"bash",
			IGNITE_IMAGE,
			"-c",
			"tail -f /dev/null",
		})
	}

}

func (runtime *Runtime) Stop() (string, error) {
	runtime.stopChan <- true
	return system.RunCommandGetResults("sudo", []string{
		runtime.Kind,
		"rm",
		"-f",
		runtime.Name,
	})
}

// create a script from the job commands
// these means we can run all commands as a single process
// that can be invoked by psrecord
// to do this - we need the commands inside the runtime as a "job.sh" file
// (so we can "bash job.sh" as the command)
// let's write our "job.sh" and copy it onto the runtime
func (runtime *Runtime) PrepareJob(
	// if this is defined then it means we are in development mode
	// and don't want to connect to the mainline ipfs DHT but
	// have a local development cluster of ipfs nodes instead
	connectToIpfsMultiaddress string,
) error {
	threadLogger := logger.LoggerWithRuntimeInfo(runtime.JobId)

	tmpFile, err := ioutil.TempFile("", "bacalhau-ignite-job.*.sh")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	// put sleep here because otherwise psrecord does not have enough time to capture metrics
	script := fmt.Sprintf("sleep 2\n%s\nsleep 2\n", strings.Join(runtime.Job.Commands[:], "\n"))
	_, err = tmpFile.WriteString(script)
	if err != nil {
		return err
	}

	threadLogger.Debug().Msgf("Script to run for job: %s", script)

	output, err := system.RunCommandGetResults("sudo", []string{
		runtime.Kind,
		"cp",
		tmpFile.Name(),
		fmt.Sprintf("%s:/job.sh", runtime.Name),
	})
	if err != nil {
		return fmt.Errorf(`Error copying script to execution location: 
Error: %+v
Output: %s`, err, output)
	}

	output, err = system.RunCommandGetResults("sudo", cleanEmpty([]string{
		runtime.Kind,
		"exec",
		runtime.Name,
		runtime.doubleDash, "ipfs", "init",
	}))
	if err != nil {
		return fmt.Errorf(`Error copying script to execution location: 
Error: %+v
Output: %s`, err, output)
	}

	if connectToIpfsMultiaddress != "" {
		output, err = system.RunCommandGetResults("sudo", cleanEmpty([]string{
			runtime.Kind,
			"exec",
			runtime.Name,
			runtime.doubleDash, "ipfs", "bootstrap", "rm", "--all",
		}))
		if err != nil {
			return fmt.Errorf(`Error rm during bootstraping ipfs to execution location: 
Error: %+v
Output: %s`, err, output)
		}
		output, err = system.RunCommandGetResults("sudo", cleanEmpty([]string{
			runtime.Kind,
			"exec",
			runtime.Name,
			runtime.doubleDash, "ipfs", "bootstrap", "add", connectToIpfsMultiaddress,
		}))
		if err != nil {
			return fmt.Errorf(`Error adding during bootstraping ipfs to execution location: 
Error: %+v
Output: %s`, err, output)
		}

	}

	command := "sudo"
	args := cleanEmpty([]string{
		runtime.Kind,
		"exec",
		runtime.Name,
		runtime.doubleDash, "ipfs", "daemon", "--mount",
	})

	cmd := exec.Command(command, args...)

	threadLogger.Debug().Msgf("Running system ipfs daemon mount: %s", cmd.String())

	logfile, err := os.OpenFile(BACALHAU_LOGFILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	cmd.Stderr = logfile
	cmd.Stdout = logfile

	go func() {
		err := cmd.Run()
		if err != nil {
			threadLogger.Debug().Msgf("Failed to run : %s", err)
		}
	}()

	go func() {
		<-runtime.stopChan

		// TODO: Should we check the result here?
		cmd.Process.Kill() // nolint
	}()

	// sleep here to give the "ipfs daemon --mount" command time to start
	time.Sleep(5 * time.Second)

	return nil
}

// TODO: mount input data files
// TODO: mount output data files
// psrecord invoke the job that we have prepared at /job.sh
// copy the psrecord metrics out of the runtime
// TODO: bunlde the results data and metrics
func (runtime *Runtime) RunJob(resultsFolder string) error {
	threadLogger := logger.LoggerWithRuntimeInfo(runtime.JobId)

	err, stdout, stderr := system.RunTeeCommand("sudo", cleanEmpty([]string{
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
