package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var JobSelectionFlags = []Definition{
	{
		FlagName:          "job-selection-data-locality",
		ConfigPath:        "job.selection.data.locality.deprecated",
		DefaultValue:      "",
		Deprecated:        true,
		DeprecatedMessage: "Locality is no longer configurable.",
	},
	{
		FlagName:     "job-selection-reject-stateless",
		ConfigPath:   "JobAdmissionControl.RejectStatelessJobs",
		DefaultValue: types2.Default.JobAdmissionControl.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "job-selection-accept-networked",
		ConfigPath:   "JobAdmissionControl.AcceptNetworkedJobs",
		DefaultValue: types2.Default.JobAdmissionControl.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "job-selection-probe-http",
		ConfigPath:   "JobAdmissionControl.ProbeHTTP",
		DefaultValue: types2.Default.JobAdmissionControl.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "job-selection-probe-exec",
		ConfigPath:   "JobAdmissionControl.ProbeExec",
		DefaultValue: types2.Default.JobAdmissionControl.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}
