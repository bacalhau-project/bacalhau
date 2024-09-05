import React from 'react';
import { Button } from "@/components/ui/button";
import { PauseIcon, PlayIcon } from 'lucide-react';
import { models_Job } from '@/lib/api/generated';

interface JobActionsProps {
  job: models_Job;
}

const JobActions: React.FC<JobActionsProps> = ({ job }) => {
  return (
    <div className="flex space-x-2">
      {job.State === 'Ok' ? (
        <Button size="sm" variant="outline">
          <PauseIcon className="h-4 w-4" />
        </Button>
      ) : (
        <Button size="sm" variant="outline">
          <PlayIcon className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
};

export default JobActions;