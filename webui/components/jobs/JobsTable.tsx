import Link from 'next/link'
import { models_Job } from '@/lib/api/generated';

import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import TruncatedTextWithTooltip from '@/components/TruncatedTextWithTooltip';
import JobStatusBadge from '@/components/jobs/JobStatusBadge';
import { formatTimestamp, getJobRunTime, shortID } from '@/lib/api/utils';
import JobEngineDisplay from "@/components/jobs/JobEngine";

interface JobsTableProps {
  jobs: models_Job[];
}

export function JobsTable({ jobs = [] }: JobsTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created At</TableHead>
          <TableHead>Run Time</TableHead>
          <TableHead>Message</TableHead>
          <TableHead>Engine</TableHead>
          <TableHead>Type</TableHead>
          <TableHead>Results Destination</TableHead>
          <TableHead>Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {jobs.map((job) => (
          <TableRow key={job.ID}>
            <TableCell><Link href={`/jobs/${job.ID}`}>{shortID(job.ID)}</Link></TableCell>
            <TableCell><JobStatusBadge status={job.State?.StateType} /></TableCell>
            <TableCell>{formatTimestamp(job.CreateTime)}</TableCell>
            <TableCell>{getJobRunTime(job)}</TableCell>
            <TableCell><TruncatedTextWithTooltip text={job.State?.Message} maxLength={30} /></TableCell>
            <TableCell> <JobEngineDisplay job={job} /></TableCell>
            <TableCell>{job.Type}</TableCell>
            <TableCell className="max-w-xs truncate">{"s3"}</TableCell>
            <TableCell>
              {/*<JobActions job={job} />*/}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
