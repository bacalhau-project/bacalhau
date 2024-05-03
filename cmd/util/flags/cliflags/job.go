package cliflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	JobNameUsageMsg = `The name to refer to this job by`

	NamespaceUsageMsg = `The namespace to associate with this job`

	TypeUsageMsg = `The type of the job (batch, ops, system, or daemon)`

	PriorityUsageMsg = `The priority of the job`

	CountUsageMsg = `The number of instances of this job to run`

	ConstraintsUsageMsg = "Selector (label query) to filter nodes on which this job can be executed, supports " +
		"'=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified " +
		"label constraints."

	LabelsUsageMsg = "List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not " +
		"matching /a-zA-Z0-9_:|-/ and all emojis will be stripped."
)

type JobSettings struct {
	Name        string
	Namespace   string
	Type        string
	Priority    int
	Count       int
	Constraints string
	Labels      map[string]string
}

func DefaultJobSettings() *JobSettings {
	return &JobSettings{
		Name:        idgen.NewJobName(),
		Namespace:   "default",
		Type:        models.JobTypeBatch,
		Priority:    0,
		Count:       1,
		Constraints: "",
		Labels:      make(map[string]string),
	}
}

func RegisterJobFlags(cmd *cobra.Command, s *JobSettings) {
	fs := pflag.NewFlagSet("job", pflag.ContinueOnError)
	fs.StringVar(&s.Name, "name", s.Name, JobNameUsageMsg)
	fs.StringVar(&s.Namespace, "namespace", s.Namespace, NamespaceUsageMsg)
	fs.StringVar(&s.Type, "type", s.Type, TypeUsageMsg)
	fs.IntVar(&s.Priority, "priority", s.Priority, PriorityUsageMsg)
	fs.IntVar(&s.Count, "count", s.Count, CountUsageMsg)
	fs.StringVar(&s.Constraints, "constraints", s.Constraints, ConstraintsUsageMsg)
	fs.StringToStringVarP(&s.Labels, "label", "l", s.Labels, LabelsUsageMsg)

	cmd.Flags().AddFlagSet(fs)
}
