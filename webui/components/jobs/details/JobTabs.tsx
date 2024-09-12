import React, { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  models_Job,
  apimodels_ListJobHistoryResponse,
  apimodels_ListJobExecutionsResponse,
} from '@/lib/api/generated'
import JobHistory from './JobHistory'
import JobExecutions from './JobExecutions'
import JobInspect from './JobInspect'
import JobLogs from './JobLogs'

const JobTabs = ({
  job,
  history,
  executions,
}: {
  job: models_Job
  history?: apimodels_ListJobHistoryResponse
  executions?: apimodels_ListJobExecutionsResponse
}) => {
  const [activeTab, setActiveTab] = useState('history')

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab}>
      <TabsList>
        <TabsTrigger value="history">History</TabsTrigger>
        <TabsTrigger value="executions">Executions</TabsTrigger>
        <TabsTrigger value="inspect">Inspect</TabsTrigger>
        <TabsTrigger value="logs">Logs</TabsTrigger>
      </TabsList>

      <TabsContent value="history">
        <JobHistory history={history} />
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
