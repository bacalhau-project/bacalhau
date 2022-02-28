package ignite

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

const IGNITE_IMAGE string = "binocarlos/bacalhau-ignite-image:v1"

type Vm struct {
	Id       string
	Name     string
	Job      *types.Job
	stopChan chan bool
}

func NewVm(job *types.Job) (*Vm, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s%s", job.Id, id.String())
	vm := &Vm{
		Id:       id.String(),
		Name:     name,
		Job:      job,
		stopChan: make(chan bool),
	}
	return vm, nil
}

// start the vm so we can exec to prepare and run the job
func (vm *Vm) Start() error {
	return system.RunCommand("sudo", []string{
		"ignite",
		"run",
		IGNITE_IMAGE,
		"--name",
		vm.Name,
		"--cpus",
		fmt.Sprintf("%d", vm.Job.Cpu),
		"--memory",
		fmt.Sprintf("%dGB", vm.Job.Memory),
		"--size",
		fmt.Sprintf("%dGB", vm.Job.Disk),
		"--ssh",
	})
}

func (vm *Vm) Stop() error {
	fmt.Printf("Stopping vm %s\n", vm.Name)
	vm.stopChan <- true
	return system.RunCommand("sudo", []string{
		"ignite",
		"rm",
		"-f",
		vm.Name,
	})
}

// create a script from the job commands
// these means we can run all commands as a single process
// that can be invoked by psrecord
// to do this - we need the commands inside the vm as a "job.sh" file
// (so we can "bash job.sh" as the command)
// let's write our "job.sh" and copy it onto the vm
func (vm *Vm) PrepareJob(
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
	script := fmt.Sprintf("sleep 2\n%s\nsleep 2\n", strings.Join(vm.Job.Commands[:], "\n"))
	_, err = tmpFile.WriteString(script)
	if err != nil {
		return err
	}
	err = system.RunCommand("sudo", []string{
		"ignite",
		"cp",
		tmpFile.Name(),
		fmt.Sprintf("%s:/job.sh", vm.Name),
	})
	if err != nil {
		return err
	}

	err = system.RunCommand("sudo", []string{
		"ignite",
		"exec",
		vm.Name,
		"ipfs init",
	})
	if err != nil {
		return err
	}

	if connectToIpfsMultiaddress != "" {
		err = system.RunCommand("sudo", []string{
			"ignite",
			"exec",
			vm.Name,
			"ipfs bootstrap rm --all",
		})
		if err != nil {
			return err
		}
		err = system.RunCommand("sudo", []string{
			"ignite",
			"exec",
			vm.Name,
			fmt.Sprintf("ipfs bootstrap add %s", connectToIpfsMultiaddress),
		})
		if err != nil {
			return err
		}
	}

	command := "sudo"
	args := []string{
		"bash", "-c",
		fmt.Sprintf("ignite exec %s -- ipfs daemon --mount &>> /var/log/bacalhau.log", vm.Name),
	}

	system.CommandLogger(command, args)

	cmd := exec.Command(command, args...)
	// XXX DANGER WILL ROBINSON: uncommenting the following lines leads to terrible deadlocks
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout

	go func() {
		err := cmd.Run()
		if err != nil {
			log.Printf("Starting ipfs daemon --mount inside the vm failed with: %s", err)
		}
	}()

	go func() {
		<-vm.stopChan
		fmt.Printf("KILLING IGNITE PID: %d\n", cmd.Process.Pid)
		cmd.Process.Signal(syscall.SIGTERM)
		//cmd.Process.Kill()
	}()

	// sleep here to give the "ipfs daemon --mount" command time to start
	time.Sleep(5 * time.Second)

	return nil
}

// TODO: mount input data files
// TODO: mount output data files
// psrecord invoke the job that we have prepared at /job.sh
// copy the psrecord metrics out of the vm
// TODO: bunlde the results data and metrics
func (vm *Vm) RunJob(resultsFolder string) error {

	err, stdout, stderr := system.RunTeeCommand("sudo", []string{
		"ignite",
		"exec",
		vm.Name,
		fmt.Sprintf("psrecord 'bash /job.sh' --log /tmp/metrics.log --plot /tmp/metrics.png --include-children"),
	})

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

	// copy the psrecord metrics out of the vm
	filesToCopy := []string{
		"metrics.log",
		"metrics.png",
	}

	for _, file := range filesToCopy {
		fmt.Printf("writing %s to %s/%s\n", file, resultsFolder, file)
		err = system.RunCommand("sudo", []string{
			"ignite",
			"cp",
			fmt.Sprintf("%s:/tmp/%s", vm.Name, file),
			fmt.Sprintf("%s/%s", resultsFolder, file),
		})
	}

	return err
}
