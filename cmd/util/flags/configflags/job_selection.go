package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var JobSelectionFlags = []Definition{
	{
		FlagName:          "job-selection-data-locality",
		ConfigPath:        "job.selection.data.locality.deprecated",
		DefaultValue:      "",
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "Locality is no longer configurable.",
	},
	{
		FlagName:     "job-selection-reject-stateless",
		ConfigPath:   cfgtypes.JobAdmissionControlRejectStatelessJobsKey,
		DefaultValue: cfgtypes.Default.JobAdmissionControl.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "job-selection-accept-networked",
		ConfigPath:   cfgtypes.JobAdmissionControlAcceptNetworkedJobsKey,
		DefaultValue: cfgtypes.Default.JobAdmissionControl.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "job-selection-probe-http",
		ConfigPath:   cfgtypes.JobAdmissionControlProbeHTTPKey,
		DefaultValue: cfgtypes.Default.JobAdmissionControl.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "job-selection-probe-exec",
		ConfigPath:   cfgtypes.JobAdmissionControlProbeExecKey,
		DefaultValue: cfgtypes.Default.JobAdmissionControl.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}
