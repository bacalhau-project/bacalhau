import React from 'react'
import { Badge } from '@/components/ui/badge'
import { models_JobStateType } from '@/lib/api/generated'
import { getJobState, getJobStateLabel } from '@/lib/api/utils'

interface StatusBadgeProps {
  status: string | number | models_JobStateType | undefined
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ status }) => {
  const jobStateType = getJobState(status)
  const label = getJobStateLabel(jobStateType)

  const getJobStateColor = (
    status: models_JobStateType | undefined
  ): string => {
    switch (status) {
      case models_JobStateType.JobStateTypeUndefined:
        return 'bg-gray-100 text-gray-800'
      case models_JobStateType.JobStateTypePending:
        return 'bg-yellow-100 text-yellow-800'
      case models_JobStateType.JobStateTypeQueued:
        return 'bg-blue-100 text-blue-800'
      case models_JobStateType.JobStateTypeRunning:
        return 'bg-purple-100 text-purple-800'
      case models_JobStateType.JobStateTypeCompleted:
        return 'bg-green-100 text-green-800'
      case models_JobStateType.JobStateTypeFailed:
        return 'bg-red-100 text-red-800'
      case models_JobStateType.JobStateTypeStopped:
        return 'bg-orange-100 text-orange-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const colorClass = getJobStateColor(jobStateType)

  return <Badge className={`text-xs ${colorClass}`}>{label}</Badge>
}

export default StatusBadge
