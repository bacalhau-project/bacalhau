'use client'
import React, { useEffect, useCallback } from 'react'
import { Orchestrator, apimodels_GetJobResponse } from '@/lib/api/generated'
import { JobInformation } from './JobInformation'
import JobActions from './JobActions'
import JobTabs from './JobTabs'
import { useApiOperation } from '@/hooks/useApiOperation'
import { ErrorDisplay } from '@/components/ErrorDisplay'
import { Skeleton } from '@/components/ui/skeleton'

const JobDetails = ({ jobId }: { jobId: string }) => {
  const {
    data: jobData,
    isLoading,
    error,
    execute,
  } = useApiOperation<apimodels_GetJobResponse>()

  const fetchJobData = useCallback(() => {
    execute(() =>
      Orchestrator.getJob({
        path: { id: jobId },
        query: {
          include: 'history,executions',
        },
        throwOnError: true,
      }).then((response) => response.data)
    )
  }, [execute, jobId])

  useEffect(() => {
    fetchJobData()
  }, [fetchJobData])

  if (isLoading) return <JobDetailsSkeleton />
  if (error) return <ErrorDisplay error={error} />
  if (!jobData || !jobData.Job) return

  const { Job, History, Executions } = jobData

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{Job.Name}</h1>
        <JobActions job={Job} onJobUpdated={fetchJobData} />
      </div>
      <JobInformation job={Job} />
      <JobTabs job={Job} history={History} executions={Executions} />
    </div>
  )
}

const JobDetailsSkeleton = () => (
  <div className="container mx-auto p-4">
    <div className="flex justify-between items-center mb-4">
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-10 w-32" />
    </div>
    <Skeleton className="h-40 w-full mb-4" />
    <Skeleton className="h-8 w-full mb-2" />
    <Skeleton className="h-[300px] w-full" />
  </div>
)

export default JobDetails
