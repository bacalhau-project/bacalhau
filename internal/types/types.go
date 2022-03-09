package types

import (
	"fmt"
	"strings"
)

type JobSpec struct {
	Id       string
	Cids     []string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}

type JobState struct {
	JobId     string
	NodeId    string
	State     string
	Status    string
	ResultCid string
}

// the view of a single job
// multiple compute nodes will be running this job
type JobData struct {
	Job   *JobSpec
	State map[string]*JobState
}

// the data structure a client can use to render a view of the state of the world
// e.g. this is used to render the CLI table and results list
type ListResponse struct {
	Jobs map[string]*JobData
}

func PrettyPrintJob(j *JobSpec) string {

	return fmt.Sprintf(`
	Id: %s
	Cids: %s
	Commands: %s
	Cpu: %d
	Memory %d
	Disk: %d
`, j.Id, strings.Join(j.Cids, ","), strings.Join(j.Commands, "', '"), j.Cpu, j.Disk, j.Memory)

}
