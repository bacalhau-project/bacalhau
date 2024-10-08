package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var JobSelectionFlags = []Definition{
	{
		FlagName:          "job-selection-data-locality",
		ConfigPath:        "job.selection.data.locality.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: FeatureDeprecatedMessage,
	},
	{
		FlagName:             "job-selection-reject-stateless",
		ConfigPath:           types.JobAdmissionControlRejectStatelessJobsKey,
		DefaultValue:         config.Default.JobAdmissionControl.RejectStatelessJobs,
		Description:          `Reject jobs that don't specify any data.`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_JOBSELECTION_REJECTSTATELESSJOBS"},
		DeprecatedMessage:    makeDeprecationMessage(types.JobAdmissionControlRejectStatelessJobsKey),
	},
	{
		FlagName:             "job-selection-accept-networked",
		ConfigPath:           types.JobAdmissionControlAcceptNetworkedJobsKey,
		DefaultValue:         config.Default.JobAdmissionControl.AcceptNetworkedJobs,
		Description:          `Accept jobs that require network access.`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_JOBSELECTION_ACCEPTNETWORKEDJOBS"},
		DeprecatedMessage:    makeDeprecationMessage(types.JobAdmissionControlAcceptNetworkedJobsKey),
	},
	{
		FlagName:             "job-selection-probe-http",
		ConfigPath:           types.JobAdmissionControlProbeHTTPKey,
		DefaultValue:         config.Default.JobAdmissionControl.ProbeHTTP,
		Description:          `Use the result of a HTTP POST to decide if we should take on the job.`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_JOBSELECTION_PROBEHTTP"},
		DeprecatedMessage:    makeDeprecationMessage(types.JobAdmissionControlProbeHTTPKey),
	},
	{
		FlagName:             "job-selection-probe-exec",
		ConfigPath:           types.JobAdmissionControlProbeExecKey,
		DefaultValue:         config.Default.JobAdmissionControl.ProbeExec,
		Description:          `Use the result of a exec an external program to decide if we should take on the job.`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_COMPUTE_JOBSELECTION_PROBEEXEC"},
		DeprecatedMessage:    makeDeprecationMessage(types.JobAdmissionControlProbeExecKey),
	},
}
