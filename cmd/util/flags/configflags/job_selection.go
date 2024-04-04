package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

//
// Requester Job Selection Policy
//
// TODO: https://github.com/bacalhau-project/bacalhau/issues/2929
// The flags are currently settable on the requester, however the requester does not support a JobSelectionPolicy
// setting the flags is a noop until the above issue is fixed.

var JobSelectionFlags = []Definition{
	{
		FlagName:     "job-selection-data-locality",
		ConfigPath:   types.NodeComputeJobSelectionPolicyLocality,
		DefaultValue: Default.Node.Compute.JobSelection.Policy.Locality,
		Description:  `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		FlagName:     "job-selection-reject-stateless",
		ConfigPath:   types.NodeComputeJobSelectionPolicyRejectStatelessJobs,
		DefaultValue: Default.Node.Compute.JobSelection.Policy.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "job-selection-accept-networked",
		ConfigPath:   types.NodeComputeJobSelectionPolicyAcceptNetworkedJobs,
		DefaultValue: Default.Node.Compute.JobSelection.Policy.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "job-selection-probe-http",
		ConfigPath:   types.NodeComputeJobSelectionPolicyProbeHTTP,
		DefaultValue: Default.Node.Compute.JobSelection.Policy.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "job-selection-probe-exec",
		ConfigPath:   types.NodeComputeJobSelectionPolicyProbeExec,
		DefaultValue: Default.Node.Compute.JobSelection.Policy.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}
