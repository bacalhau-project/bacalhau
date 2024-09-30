import React, { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  models_Job,
  apimodels_ListJobExecutionsResponse,
} from '@/lib/api/generated'
import JobHistoryContainer from './JobHistoryContainer'
import JobExecutions from './JobExecutions'
import JobInspect from './JobInspect'
import JobLogs from './JobLogs'

const JobTabs = ({
  job,
  executions,
}: {
  job: models_Job
  executions?: apimodels_ListJobExecutionsResponse
}) => {
  const [activeTab, setActiveTab] = useState('history')

  const executionCount = executions?.Items?.length ?? 0

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab}>
      <TabsList>
        <TabsTrigger value="history">History</TabsTrigger>
        <TabsTrigger value="executions">
          Executions
          {executionCount > 0 && (
            <span className="ml-2 inline-flex items-center justify-center w-6 h-6 text-xs font-bold text-white bg-gray-600 rounded-full">
              {executionCount}
            </span>
          )}
        </TabsTrigger>
        <TabsTrigger value="inspect">Inspect</TabsTrigger>
        <TabsTrigger value="logs">Logs</TabsTrigger>
      </TabsList>

      <TabsContent value="history">
        <JobHistoryContainer jobId={job.ID!} />
      </TabsContent>

      <TabsContent value="executions">
        <JobExecutions executions={executions} />
      </TabsContent>

      <TabsContent value="inspect">
        <JobInspect job={job} />
      </TabsContent>

      <TabsContent value="logs">
        <JobLogs jobId={job.ID} />
      </TabsContent>
    </Tabs>
  )
}

export default JobTabs
