'use client'

import React, { useState, useEffect, useCallback } from 'react'
import { NodesTable } from './NodesTable'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { models_NodeState, OrchestratorService } from '@/lib/api/generated'
import { useApi } from '@/app/providers/ApiProvider'
import { useRefreshContent } from '@/hooks/useRefreshContent'
import { RefreshCw } from 'lucide-react'

export function NodesOverview() {
  const [nodes, setNodes] = useState<models_NodeState[]>([])
  const [search, setSearch] = useState('')
  const { isInitialized } = useApi()
  const [pageSize, setPageSize] = useState(20)
  const [pageIndex, setPageIndex] = useState(0)
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [isRefreshDisabled, setIsRefreshDisabled] = useState(false)

  const fetchNodes = useCallback(async () => {
    if (!isInitialized) return

    try {
      const response = await OrchestratorService.orchestratorListNodes(
        pageSize,
        pageIndex === 0 ? undefined : nextToken,
        true, // reverse
        undefined, // orderBy
        undefined, // filterApproval
        undefined // filterStatus
      )
      setNodes(response.Nodes ?? [])
      setNextToken(response.NextToken)
    } catch (error) {
      console.error('Error fetching nodes:', error)
      setNodes([])
    }
  }, [isInitialized, pageSize, pageIndex, nextToken])

  useEffect(() => {
    fetchNodes()
  }, [fetchNodes])

  const handleRefresh = useCallback(() => {
    setIsRefreshDisabled(true)
    setPageIndex(0)
    setNextToken(undefined)
    fetchNodes().then(() => {
      setTimeout(() => setIsRefreshDisabled(false), 1000)
    })
  }, [fetchNodes])

  useRefreshContent('nodes', handleRefresh)

  const filteredNodes = nodes.filter(
    (node) => node.Info?.NodeID?.toLowerCase().includes(search.toLowerCase()) ?? false
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
      <h1 className="text-3xl font-bold mb-8">Nodes</h1>
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center space-x-2">
          <Input
            className="w-80"
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter nodes..."
          />
          <Button
            onClick={handleRefresh}
            disabled={isRefreshDisabled}
            variant="outline"
            size="icon"
            aria-label="Refresh nodes"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <NodesTable
        nodes={filteredNodes}
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