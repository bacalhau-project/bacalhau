package runtime

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

const IGNITE_IMAGE string = "binocarlos/bacalhau-ignite-image:v1"
const BACALHAU_LOGFILE = "/tmp/bacalhau.log"

type Runtime struct {
	Id       string
	Name     string
	Job      *types.Job
	stopChan chan bool
}

func NewRuntime(job *types.Job) (*Runtime, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s%s", job.Id, id.String())
	runtime := &Runtime{
		Id:       id.String(),
		Name:     name,
		Job:      job,
		stopChan: make(chan bool),
	}
	return runtime, nil
}

// start the runtime so we can exec to prepare and run the job
func (runtime *Runtime) Start() error {
	return system.RunCommand("sudo", []string{
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
}

func (runtime *Runtime) Stop() error {
	runtime.stopChan <- true
	return system.RunCommand("sudo", []string{
		"ignite",
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
	err = system.RunCommand("sudo", []string{
		"ignite",
		"cp",
		tmpFile.Name(),
		fmt.Sprintf("%s:/job.sh", runtime.Name),
	})
	if err != nil {
		return err
	}

	err = system.RunCommand("sudo", []string{
		"ignite",
		"exec",
		runtime.Name,
		"ipfs init",
	})
	if err != nil {
		return err
	}

	if connectToIpfsMultiaddress != "" {
		err = system.RunCommand("sudo", []string{
			"ignite",
			"exec",
			runtime.Name,
			"ipfs bootstrap rm --all",
		})
		if err != nil {
			return err
		}
		err = system.RunCommand("sudo", []string{
			"ignite",
			"exec",
			runtime.Name,
			fmt.Sprintf("ipfs bootstrap add %s", connectToIpfsMultiaddress),
		})
		if err != nil {
			return err
		}
	}

	command := "sudo"
	args := []string{
		"ignite",
		"exec",
		runtime.Name,
		"ipfs daemon --mount",
	}

	system.CommandLogger(command, args)

	cmd := exec.Command(command, args...)
	logfile, err := os.OpenFile(BACALHAU_LOGFILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	cmd.Stderr = logfile
	cmd.Stdout = logfile

	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout

	go func() {
		err := cmd.Run()
		if err != nil {
			log.Printf("Starting ipfs daemon --mount inside the runtime failed with: %s", err)
		}
	}()

	go func() {
		<-runtime.stopChan
		cmd.Process.Kill()
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

	err, stdout, stderr := system.RunTeeCommand("sudo", []string{
		"ignite",
		"exec",
		runtime.Name,
		"psrecord 'bash /job.sh' --log /tmp/metrics.log --plot /tmp/metrics.png --include-children",
	})
	if err != nil {
		return err
	}

	// write the command stdout & stderr to the results dir
	fmt.Printf("writing stdout to %s/stdout.log\n", resultsFolder)
	err = os.WriteFile(fmt.Sprintf("%s/stdout.log", resultsFolder), stdout.Bytes(), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("writing stderr to %s/stderr.log\n", resultsFolder)
	err = os.WriteFile(fmt.Sprintf("%s/stderr.log", resultsFolder), stderr.Bytes(), 0644)
	if err != nil {
		return err
	}

	// copy the psrecord metrics out of the runtime
	filesToCopy := []string{
		"metrics.log",
		"metrics.png",
	}

	for _, file := range filesToCopy {
		fmt.Printf("writing %s to %s/%s\n", file, resultsFolder, file)
		err = system.RunCommand("sudo", []string{
			"ignite",
			"cp",
			fmt.Sprintf("%s:/tmp/%s", runtime.Name, file),
			fmt.Sprintf("%s/%s", resultsFolder, file),
		})
	}

	return err
}
