package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var JobSelectionFlags = []Definition{
	//
	// Requester Job Selection Policy
	//
	// TODO: https://github.com/bacalhau-project/bacalhau/issues/2929
	// The flags are currently settable on the requester, however the requester does not support a JobSelectionPolicy
	// setting the flags is a noop until the above issue is fixed.
	{
		FlagName:     "requester-job-selection-data-locality",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyLocality,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.Locality,
		Description:  `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		FlagName:     "requester-job-selection-reject-stateless",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyRejectStatelessJobs,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "requester-job-selection-accept-networked",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyAcceptNetworkedJobs,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "requester-job-selection-probe-http",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyProbeHTTP,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "requester-job-selection-probe-exec",
		ConfigPath:   types.NodeRequesterJobSelectionPolicyProbeExec,
		DefaultValue: Default.Node.Requester.JobSelectionPolicy.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
	//
	// Compute Job Selection Policy
	{
		FlagName:     "compute-job-selection-data-locality",
		ConfigPath:   types.NodeComputeJobSelectionLocality,
		DefaultValue: Default.Node.Compute.JobSelection.Locality,
		Description:  `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		FlagName:     "compute-job-selection-reject-stateless",
		ConfigPath:   types.NodeComputeJobSelectionRejectStatelessJobs,
		DefaultValue: Default.Node.Compute.JobSelection.RejectStatelessJobs,
		Description:  `Reject jobs that don't specify any data.`,
	},
	{
		FlagName:     "compute-job-selection-accept-networked",
		ConfigPath:   types.NodeComputeJobSelectionAcceptNetworkedJobs,
		DefaultValue: Default.Node.Compute.JobSelection.AcceptNetworkedJobs,
		Description:  `Accept jobs that require network access.`,
	},
	{
		FlagName:     "compute-job-selection-probe-http",
		ConfigPath:   types.NodeComputeJobSelectionProbeHTTP,
		DefaultValue: Default.Node.Compute.JobSelection.ProbeHTTP,
		Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		FlagName:     "compute-job-selection-probe-exec",
		ConfigPath:   types.NodeComputeJobSelectionProbeExec,
		DefaultValue: Default.Node.Compute.JobSelection.ProbeExec,
		Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}
