package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var JobSelectionFlags = []Definition{
	{
		FlagName:     "job-selection-data-locality",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyLocality,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.Locality,
		Description:  `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		FlagName:     "job-selection-reject-stateless",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyRejectStatelessJobs,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "job-selection-accept-networked",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyAcceptNetworkedJobs,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "job-selection-probe-http",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyProbeHTTP,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "job-selection-probe-exec",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyProbeExec,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}
