import React from 'react'
import { Card, CardHeader, CardContent, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { apimodels_ListJobExecutionsResponse } from '@/lib/api/generated'
import {
  formatTimestamp,
  getExecutionDesiredStateLabel,
  getExecutionStateLabel,
  getJobStateLabel,
  shortID,
} from '@/lib/api/utils'

const JobExecutions = ({
  executions,
}: {
  executions?: apimodels_ListJobExecutionsResponse
}) => (
  <Card>
    <CardContent className="pt-6">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Created Time</TableHead>
            <TableHead>Modified Time</TableHead>
            <TableHead>ID</TableHead>
            <TableHead>Node ID</TableHead>
            <TableHead>State</TableHead>
            <TableHead>Desired State</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {executions?.Items?.map((execution) => (
            <TableRow key={execution.ID}>
              <TableCell>
                {formatTimestamp(execution.CreateTime, true)}
              </TableCell>
              <TableCell>
                {formatTimestamp(execution.ModifyTime, true)}
              </TableCell>
              <TableCell>{shortID(execution.ID)}</TableCell>
              <TableCell>{shortID(execution.NodeID)}</TableCell>
              <TableCell>
                {getExecutionStateLabel(execution.ComputeState?.StateType)}
              </TableCell>
              <TableCell>
                {getExecutionDesiredStateLabel(
                  execution.DesiredState?.StateType
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </CardContent>
  </Card>
)

export default JobExecutions
