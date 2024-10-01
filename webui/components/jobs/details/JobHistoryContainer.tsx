import React, { useState, useCallback, useEffect } from 'react'
import {
  Orchestrator,
  apimodels_ListJobHistoryResponse,
} from '@/lib/api/generated'
import JobHistory from './JobHistory'

const JobHistoryContainer = ({ jobId }: { jobId: string }) => {
  const [history, setHistory] = useState<
    apimodels_ListJobHistoryResponse | undefined
  >()
  const [pageSize, setPageSize] = useState(100)
  const [pageIndex, setPageIndex] = useState(0)
  const [tokens, setTokens] = useState<(string | undefined)[]>([undefined])
  const [isLoading, setIsLoading] = useState(false)

  const fetchHistory = useCallback(async () => {
    setIsLoading(true)
    try {
      const response = await Orchestrator.listHistory({
        path: { id: jobId },
        query: {
          limit: pageSize,
          next_token: tokens[pageIndex],
        },
        throwOnError: true,
      })
      setHistory(response.data)
      if (response.data.NextToken && pageIndex === tokens.length - 1) {
        setTokens([...tokens, response.data.NextToken])
      }
    } catch (error) {
      console.error('Error fetching job history:', error)
      setHistory(undefined)
    } finally {
      setIsLoading(false)
    }
  }, [jobId, pageSize, pageIndex, tokens])

  useEffect(() => {
    fetchHistory()
  }, [fetchHistory])

  const handlePreviousPage = () => {
    if (pageIndex > 0) {
      setPageIndex(pageIndex - 1)
    }
  }

  const handleNextPage = () => {
    if (pageIndex < tokens.length - 1) {
      setPageIndex(pageIndex + 1)
    }
  }

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize)
    setPageIndex(0)
    setTokens([undefined])
  }

  return (
    <JobHistory
      history={history}
      isLoading={isLoading}
      pageSize={pageSize}
      pageIndex={pageIndex}
      onPreviousPage={handlePreviousPage}
      onNextPage={handleNextPage}
      onPageSizeChange={handlePageSizeChange}
      hasNextPage={pageIndex < tokens.length - 1}
    />
  )
}

export default JobHistoryContainer
