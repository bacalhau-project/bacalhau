import React from 'react'
import { Card, CardHeader, CardContent, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { models_Job } from '@/lib/api/generated'
import { formatTimestamp, getJobRunTime } from '@/lib/api/utils'
import JobStatusBadge from '@/components/jobs/JobStatusBadge'
import JobEngineDisplay from '@/components/jobs/JobEngine'
import { JobLabels } from '@/components/jobs/JobLabels'

interface JobInformationProps {
  job: models_Job
}

const JobInformation: React.FC<JobInformationProps> = ({ job }) => (
  <Card className="mb-4">
    <CardContent className="py-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <InfoItem label="Name">{job.Name}</InfoItem>
          <InfoItem label="ID">{job.ID}</InfoItem>
          <InfoItem label="Namespace">{job.Namespace}</InfoItem>
          <InfoItem label="Type">{job.Type}</InfoItem>
          <InfoItem label="State">
            <JobStatusBadge status={job.State?.StateType} />
          </InfoItem>
        </div>
        <div className="space-y-2">
          <InfoItem label="Created">
            {formatTimestamp(job.CreateTime, true)}
          </InfoItem>
          <InfoItem label="Modified">
            {formatTimestamp(job.ModifyTime, true)}
          </InfoItem>
          <InfoItem label="Run Time">{getJobRunTime(job)}</InfoItem>
          <InfoItem label="Engine">
            <JobEngineDisplay job={job} />
          </InfoItem>
          {job.Labels && Object.keys(job.Labels).length > 0 && (
            <InfoItem label="Labels">
              <JobLabels labels={job.Labels} />
            </InfoItem>
          )}
        </div>
      </div>

      {job.State?.Message && (
        <div className="mt-3">
          <InfoItem label="Message">{job.State.Message}</InfoItem>
        </div>
      )}
    </CardContent>
  </Card>
)

interface InfoItemProps {
  label: string
  children: React.ReactNode
}

const InfoItem: React.FC<InfoItemProps> = ({ label, children }) => (
  <div className="grid grid-cols-[120px,1fr] items-center py-1">
    <span className="text-sm  ont-semibold text-gray-500">{label}:</span>
    <span className="text-sm text-gray-900">{children}</span>
  </div>
)

export { JobInformation }
