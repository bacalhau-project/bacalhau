'use client'

import React, { useState, useEffect } from 'react'
import { JobsTable } from './JobsTable'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { models_Job, OrchestratorService } from '@/lib/api/generated'
import { useApi } from '@/app/providers/ApiProvider'

export function JobsOverview() {
  const [jobs, setJobs] = useState<models_Job[]>([])
  const [search, setSearch] = useState('')
  const { isInitialized } = useApi()
  const [pageSize, setPageSize] = useState(10)
  const [pageIndex, setPageIndex] = useState(0)
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)

  useEffect(() => {
    async function fetchJobs() {
      if (!isInitialized) return

      try {
        const response = await OrchestratorService.orchestratorListJobs(
          undefined, // namespace
          pageSize,
          pageIndex === 0 ? undefined : nextToken,
          true, // reverse
          undefined // orderBy
        )
        setJobs(response.Items ?? [])
        setNextToken(response.NextToken)
      } catch (error) {
        console.error('Error fetching jobs:', error)
        setJobs([])
      }
    }

    fetchJobs()
  }, [isInitialized, pageSize, pageIndex])

  const filteredJobs = jobs.filter(
    (job) =>
      (job.ID?.toLowerCase().includes(search.toLowerCase()) ?? false) ||
      (job.Name?.toLowerCase().includes(search.toLowerCase()) ?? false)
  )

  const handlePreviousPage = () => {
    if (pageIndex > 0) {
      setPageIndex(pageIndex - 1)
    }
  }

  const handleNextPage = () => {
    if (nextToken) {
      setPageIndex(pageIndex + 1)
    }
  }

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize)
    setPageIndex(0)
    setNextToken(undefined)
  }

  return (
    <div className="container mx-auto py-8">
      <h1 className="text-3xl font-bold mb-8">Jobs overview</h1>
      <div className="flex justify-between items-center mb-6">
        <Input
          className="max-w-sm"
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Filter jobs..."
        />
        <Button>Submit Job</Button>
      </div>
      <JobsTable
        jobs={filteredJobs}
        pageSize={pageSize}
        setPageSize={handlePageSizeChange}
        pageIndex={pageIndex}
        onPreviousPage={handlePreviousPage}
        onNextPage={handleNextPage}
        hasNextPage={!!nextToken}
      />
    </div>
  )
}
