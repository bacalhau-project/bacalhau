package cliflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type JobSettings struct {
	name        string
	namespace   string
	jobType     string
	priority    int
	count       int
	constraints string
	labels      map[string]string

	// TODO(forrest): remove these fields and their usage when we complete deprecation of legacy flag names.
	// tracked via https://github.com/bacalhau-project/bacalhau/issues/3838
	legacy *LegacyJobFlags
	// we hold a reference to the command so we may check if users provided legacy or new flag names.
	cmd *cobra.Command
}

func (j *JobSettings) Name() string {
	return j.name
}

func (j *JobSettings) Namespace() string {
	return j.namespace
}

func (j *JobSettings) Type() string {
	if j.cmd.Flags().Changed("target") {
		jobType := models.JobTypeBatch
		if j.legacy.targetingMode == model.TargetAll {
			jobType = models.JobTypeOps
		}
		return jobType
	}
	return j.jobType
}

func (j *JobSettings) Priority() int {
	return j.priority
}

func (j *JobSettings) Count() int {
	if j.cmd.Flags().Changed("concurrency") {
		return j.legacy.concurrency
	}
	return j.count
}

func (j *JobSettings) Constraints() ([]*models.LabelSelectorRequirement, error) {
	if j.cmd.Flags().Changed("selector") {
		req, err := parse.NodeSelector(j.legacy.selectors)
		if err != nil {
			return nil, err
		}
		out := make([]*models.LabelSelectorRequirement, 0, len(req))
		for _, c := range req {
			tmp := models.LabelSelectorRequirement(c)
			out = append(out, &tmp)
		}
		return out, nil
	}
	return parse.NodeSelectorV2(j.constraints)
}

// TODO(forrest): based on a conversation with walid we should be returning an error here if at anypoint if a label
// if provided that is invalid. We cannont remove them as we did previously.
func (j *JobSettings) Labels() (map[string]string, error) {
	rawLabels := j.labels
	s := labels.Set(rawLabels)
	if _, err := s.AsValidatedSelector(); err != nil {
		return nil, err
	}

	return rawLabels, nil
}

type LegacyJobFlags struct {
	// Deprecated: use `JobSettings.jobType`
	targetingMode model.TargetingMode
	// Deprecated: use 'JobSettings.constraints'
	selectors string
	//Deprecated: use `JobSettings.count`
	concurrency int
}

func DefaultJobSettings() *JobSettings {
	return &JobSettings{
		name:        "",
		namespace:   models.DefaultNamespace,
		jobType:     models.JobTypeBatch,
		priority:    0,
		count:       1,
		constraints: "",
		labels:      make(map[string]string),

		legacy: &LegacyJobFlags{
			targetingMode: model.TargetAny,
			selectors:     "",
			concurrency:   1,
		},
	}
}

func RegisterJobFlags(cmd *cobra.Command, s *JobSettings) {
	s.cmd = cmd
	fs := pflag.NewFlagSet("job", pflag.ContinueOnError)
	fs.StringVar(&s.name, "name", s.name,
		`The name to refer to this job by.`)

	fs.StringVar(&s.namespace, "namespace", s.namespace, `The namespace to associate with this job.`)

	fs.IntVar(&s.priority, "priority", s.priority, `The priority of the job.`)

	fs.StringToStringVarP(&s.labels, "labels", "l", s.labels,
		`List of labels for the job. Enter multiple in the format '-labels env=prod -label region=earth'.
Valid label keys must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character.
Valid label values must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end 
with an alphanumeric character.`)

	//
	// Deprecation of legacy flags tracked via https://github.com/bacalhau-project/bacalhau/issues/3838
	//

	// NB(forrest): the `count` flag is replacing `concurrency`. Hide the `concurrency` flag and add deprecation notice.
	fs.IntVar(&s.count, "count", s.count, `How many nodes should run the job.`)

	fs.IntVar(&s.legacy.concurrency, "concurrency", s.legacy.concurrency,
		`How many nodes should run the job`)

	if err := fs.MarkHidden("concurrency"); err != nil {
		panic(err)
	}
	if err := fs.MarkDeprecated("concurrency", "use --count"); err != nil {
		panic(err)
	}

	// NB(forrest): the `type` flag is replacing `targeting`. Hide the `targeting` flag and add deprecation notice.
	fs.StringVar(&s.jobType, "type", s.jobType,
		`The type of the job (batch, ops, system, or daemon).`)

	// deprecated
	fs.Var(flags.TargetingFlag(&s.legacy.targetingMode), "target",
		`Whether to target the minimum number of matching nodes ("any") (default) or all matching nodes ("all").`)

	if err := fs.MarkHidden("target"); err != nil {
		panic(err)
	}
	if err := fs.MarkDeprecated("target", "use --type"); err != nil {
		panic(err)
	}

	// NB(forrest): the `constraints` flag is replacing `selector` flag. Hide the `selector` flag and add deprecation notice.
	fs.StringVarP(&s.constraints, "constraints", "c", s.constraints,
		`Selector (label query) to filter nodes on which this job can be executed.
Supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2).
Matching objects must satisfy all of the specified label constraints.`)

	// deprecated
	fs.StringVarP(&s.legacy.selectors, "selector", "s", s.legacy.selectors,
		`Selector (label query) to filter nodes on which this job can be executed.
Supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). 
Matching objects must satisfy all of the specified label constraints.`)

	if err := fs.MarkHidden("selector"); err != nil {
		panic(err)
	}
	if err := fs.MarkDeprecated("selector", "use --constraints"); err != nil {
		panic(err)
	}

	cmd.Flags().AddFlagSet(fs)
	// NB(forrest): don't allow the legacy flag name to be used together with the new flag name.
	cmd.MarkFlagsMutuallyExclusive("count", "concurrency")
	cmd.MarkFlagsMutuallyExclusive("selector", "constraints")
	cmd.MarkFlagsMutuallyExclusive("target", "type")
	//NB(forrest): we require the name with the type.
	// TODO (forrest): FOR REVIEW this ais a question for PRODUCT cc Aronchick
	// - do we want to support type? do we want a dtype flag for each kind of job? --type-ops, --type-daemon, etc?
	//cmd.MarkFlagsRequiredTogether("type", "name")
}
