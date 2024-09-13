'use client'

import React, { useState, useEffect, useCallback } from 'react'
import { JobsTable } from './JobsTable'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { models_Job, OrchestratorService } from '@/lib/api/generated'
import { useApi } from '@/app/providers/ApiProvider'
import { useRefreshContent } from '@/hooks/useRefreshContent'
import { RefreshCw, Plus } from 'lucide-react'

export function JobsOverview() {
  const [jobs, setJobs] = useState<models_Job[]>([])
  const [search, setSearch] = useState('')
  const { isInitialized } = useApi()
  const [pageSize, setPageSize] = useState(10)
  const [pageIndex, setPageIndex] = useState(0)
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [isRefreshDisabled, setIsRefreshDisabled] = useState(false)

  const fetchJobs = useCallback(async () => {
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
  }, [isInitialized, pageSize, pageIndex, nextToken])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  const handleRefresh = useCallback(() => {
    setIsRefreshDisabled(true)
    setPageIndex(0)
    setNextToken(undefined)
    fetchJobs().then(() => {
      // Re-enable the refresh button after a short delay
      setTimeout(() => setIsRefreshDisabled(false), 1000)
    })
  }, [fetchJobs])

  // Use the custom hook
  useRefreshContent('jobs', handleRefresh)

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
    <div className="container mx-auto">
      <h1 className="text-3xl font-bold mb-8">Jobs</h1>
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center space-x-2">
          <Input
            className="w-80"
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter jobs..."
          />
          <Button
            onClick={handleRefresh}
            disabled={isRefreshDisabled}
            variant="outline"
            size="icon"
            aria-label="Refresh jobs"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
        {/*TODO: implement submit job*/}
        {/*<Button className="space-x-2">*/}
        {/*  <Plus className="h-4 w-4" />*/}
        {/*  <span>Submit Job</span>*/}
        {/*</Button>*/}
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
