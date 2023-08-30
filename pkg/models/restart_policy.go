//go:generate stringer -type=RestartPolicyType --trimprefix=RestartPolicy --output restart_policy_string.go
package models

type RestartPolicyType int

const (
	restartPolicyUndefined RestartPolicyType = iota

	// Attempt to recover the current task if the compute node believes it was
	// executing during a compute node restart
	RestartPolicyRecover

	// When restarting the compute node fail the task
	RestartPolicyFail
)

func NewRestartPolicy(typ string) RestartPolicyType {
	switch typ {
	case JobTypeService, JobTypeDaemon:
		return RestartPolicyRecover
	default: // JobTypeBatch, JobTypeOps
		return RestartPolicyFail
	}
}
