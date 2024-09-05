import React from 'react';
import { Badge } from "@/components/ui/badge";
import { models_JobStateType } from '@/lib/api/generated';
import { getJobState, getJobStateLabel } from '@/lib/api/utils';

interface StatusBadgeProps {
  status: string | number | models_JobStateType | undefined;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ status }) => {
  const jobStateType = getJobState(status);
  const label = getJobStateLabel(jobStateType);

  const getJobStateColor = (status: models_JobStateType | undefined): string => {
    switch (status) {
      case models_JobStateType.JobStateTypeUndefined:
        return 'bg-gray-500';
      case models_JobStateType.JobStateTypePending:
        return 'bg-yellow-500';
      case models_JobStateType.JobStateTypeQueued:
        return 'bg-blue-500';
      case models_JobStateType.JobStateTypeRunning:
        return 'bg-purple-500';
      case models_JobStateType.JobStateTypeCompleted:
        return 'bg-green-500';
      case models_JobStateType.JobStateTypeFailed:
        return 'bg-red-500';
      case models_JobStateType.JobStateTypeStopped:
        return 'bg-orange-500';
      default:
        return 'bg-gray-400';
    }
  };

  const color = getJobStateColor(jobStateType);

  return (
    <Badge className={`${color} text-white`}>
      {label}
    </Badge>
  );
};

export default StatusBadge;