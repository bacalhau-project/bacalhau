import Link from 'next/link'
import { models_Job } from '@/lib/api/generated'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import TruncatedTextWithTooltip from '@/components/TruncatedTextWithTooltip'
import JobStatusBadge from '@/components/jobs/JobStatusBadge'
import { formatTimestamp, getJobRunTime } from '@/lib/api/utils'
import JobEngineDisplay from '@/components/jobs/JobEngine'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface JobsTableProps {
  jobs: models_Job[]
  pageSize: number
  setPageSize: (size: number) => void
  pageIndex: number
  onPreviousPage: () => void
  onNextPage: () => void
  hasNextPage: boolean
}

export function JobsTable({
  jobs = [],
  pageSize,
  setPageSize,
  pageIndex,
  onPreviousPage,
  onNextPage,
  hasNextPage,
}: JobsTableProps) {
  return (
    <div>
      <Table>
        <TableHeader className="bg-muted/50">
          <TableRow>
            <TableHead className="p-3 w-80">ID</TableHead>
            <TableHead className="w-32">Status</TableHead>
            <TableHead className="w-40">Created At</TableHead>
            <TableHead className="w-28">Run Time</TableHead>
            <TableHead className="w-80">Message</TableHead>
            <TableHead className="w-28">Engine</TableHead>
            <TableHead className="w-28">Type</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {jobs.map((job) => (
            <TableRow key={job.ID}>
              <TableCell className="p-3">
                <Link href={`/jobs?id=${job.ID}`}>
                  <TruncatedTextWithTooltip text={job.Name} maxLength={25} />
                </Link>
              </TableCell>
              <TableCell>
                <JobStatusBadge status={job.State?.StateType} />
              </TableCell>
              <TableCell>{formatTimestamp(job.CreateTime)}</TableCell>
              <TableCell>{getJobRunTime(job)}</TableCell>
              <TableCell>
                <TruncatedTextWithTooltip
                  text={job.State?.Message}
                  maxLength={50}
                />
              </TableCell>
              <TableCell>
                <JobEngineDisplay job={job} />
              </TableCell>
              <TableCell>{job.Type}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="flex items-center justify-between space-x-2 py-4">
        <div className="flex items-center space-x-2">
          <p className="text-sm font-medium">Jobs per page</p>
          <Select
            value={`${pageSize}`}
            onValueChange={(value) => setPageSize(Number(value))}
          >
            <SelectTrigger className="h-8 w-[70px]">
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {[10, 20, 30, 40, 50].map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onPreviousPage}
            disabled={pageIndex === 0}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onNextPage}
            disabled={!hasNextPage}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
