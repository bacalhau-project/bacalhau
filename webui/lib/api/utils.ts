import {
  models_Job,
  models_JobStateType,
  models_ExecutionStateType,
  models_ExecutionDesiredStateType,
  models_NodeState,
} from './generated'
import { formatDuration, normalizeTimestamp } from '@/lib/time'

export function getJobState(
  status: string | number | undefined
): models_JobStateType | undefined {
  if (typeof status === 'string') {
    const enumKey = `JobStateType${status}` as keyof typeof models_JobStateType
    return models_JobStateType[enumKey]
  } else if (typeof status === 'number') {
    return status as models_JobStateType
  }
  return undefined
}

export function getJobStateLabel(
  status: models_JobStateType | undefined
): string {
  switch (status) {
    case models_JobStateType.JobStateTypeUndefined:
      return 'Undefined'
    case models_JobStateType.JobStateTypePending:
      return 'Pending'
    case models_JobStateType.JobStateTypeQueued:
      return 'Queued'
    case models_JobStateType.JobStateTypeRunning:
      return 'Running'
    case models_JobStateType.JobStateTypeCompleted:
      return 'Completed'
    case models_JobStateType.JobStateTypeFailed:
      return 'Failed'
    case models_JobStateType.JobStateTypeStopped:
      return 'Stopped'
    default:
      return 'Unknown'
  }
}

export function getExecutionState(
  status: string | number | undefined
): models_ExecutionStateType | undefined {
  if (typeof status === 'string') {
    const enumKey =
      `ExecutionStateType${status}` as keyof typeof models_ExecutionStateType
    return models_ExecutionStateType[enumKey]
  } else if (typeof status === 'number') {
    return status as models_ExecutionStateType
  }
  return undefined
}

export function getExecutionStateLabel(
  status: models_ExecutionStateType | undefined
): string {
  switch (status) {
    case models_ExecutionStateType.ExecutionStateUndefined:
      return 'Undefined'
    case models_ExecutionStateType.ExecutionStateNew:
      return 'New'
    case models_ExecutionStateType.ExecutionStateAskForBid:
      return 'AskForBid'
    case models_ExecutionStateType.ExecutionStateAskForBidAccepted:
      return 'AskForBidAccepted'
    case models_ExecutionStateType.ExecutionStateAskForBidRejected:
      return 'AskForBidRejected'
    case models_ExecutionStateType.ExecutionStateBidAccepted:
      return 'BidAccepted'
    case models_ExecutionStateType.ExecutionStateBidRejected:
      return 'BidRejected'
    case models_ExecutionStateType.ExecutionStateCompleted:
      return 'Completed'
    case models_ExecutionStateType.ExecutionStateFailed:
      return 'Failed'
    case models_ExecutionStateType.ExecutionStateCancelled:
      return 'Cancelled'
    default:
      return 'Unknown'
  }
}

export function getExecutionDesiredState(
  status: string | number | undefined
): models_ExecutionDesiredStateType | undefined {
  if (typeof status === 'string') {
    const enumKey =
      `ExecutionDesiredStateType${status}` as keyof typeof models_ExecutionDesiredStateType
    return models_ExecutionDesiredStateType[enumKey]
  } else if (typeof status === 'number') {
    return status as models_ExecutionDesiredStateType
  }
  return undefined
}

export function getExecutionDesiredStateLabel(
  status: models_ExecutionDesiredStateType | undefined
): string {
  switch (status) {
    case models_ExecutionDesiredStateType.ExecutionDesiredStatePending:
      return 'Pending'
    case models_ExecutionDesiredStateType.ExecutionDesiredStateRunning:
      return 'Running'
    case models_ExecutionDesiredStateType.ExecutionDesiredStateStopped:
      return 'Stopped'
    default:
      return 'Unknown'
  }
}

export function isTerminalJobState(
  status: models_JobStateType | undefined
): boolean {
  status = getJobState(status)
  return (
    status === models_JobStateType.JobStateTypeCompleted ||
    status === models_JobStateType.JobStateTypeFailed ||
    status === models_JobStateType.JobStateTypeStopped
  )
}

// returns short job, execution and nodeIDs.
// e.g. shortID('j-3514ff75-c6f6-4380-8dc6-88f31ce7a1b3') => 'j-3514ff75'
// e.g. shortID('e-3514ff75-c6f6-4380-8dc6-88f31ce7a1b3') => 'e-3514ff75'
// e.g. shortID('n-3514ff75-c6f6-4380-8dc6-88f31ce7a1b3') => 'n-3514ff75'
export function shortID(id: string | undefined): string {
  if (!id) return 'N/A'
  return id.split('-').slice(0, 2).join('-')
}

export const getNodeConnectionStatus = (node: models_NodeState): string => {
  // Use type assertion to tell TypeScript that Connection might be a string
  const connection = node.Connection as unknown as string

  if (connection) {
    return connection.toUpperCase()
  }

  // Fallback for unexpected cases
  return 'UNKNOWN'
}

export const getNodeMembershipStatus = (node: models_NodeState): string => {
  // Use type assertion to tell TypeScript that Membership might be a string
  const membership = node.Membership as unknown as string
  if (membership) {
    return membership.toUpperCase()
  }

  // Fallback for unexpected cases
  return 'UNKNOWN'
}

export const getNodeType = (node: models_NodeState): string => {
  const nodeType = node.Info?.NodeType as unknown as string
  if (nodeType) {
    if (nodeType === 'Requester') {
      return 'Orchestrator'
    }
    return nodeType
  }
  return 'Unknown'
}

export function getJobRunTime(job: models_Job): string {
  if (!job.State?.StateType || !job.CreateTime) {
    return 'N/A'
  }

  const createTime = normalizeTimestamp(job.CreateTime)
  const endTime =
    isTerminalJobState(job.State.StateType) && job.ModifyTime
      ? normalizeTimestamp(job.ModifyTime)
      : Date.now()

  const durationMs = endTime - createTime
  return formatDuration(durationMs)
}
