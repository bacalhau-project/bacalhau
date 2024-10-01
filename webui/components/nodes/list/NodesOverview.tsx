'use client'

import React, { useState, useEffect, useCallback } from 'react'
import { NodesTable } from './NodesTable'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Orchestrator, models_NodeState } from '@/lib/api/generated'
import { useApi } from '@/app/providers/ApiProvider'
import { useRefreshContent } from '@/hooks/useRefreshContent'
import { RefreshCw, Server } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { getNodeConnectionStatus } from '@/lib/api/utils'

export function NodesOverview() {
  const [nodes, setNodes] = useState<models_NodeState[]>([])
  const [connectedNodes, setConnectedNodes] = useState<number>(0)
  const [search, setSearch] = useState('')
  const { isInitialized } = useApi()
  const [pageSize, setPageSize] = useState(20)
  const [pageIndex, setPageIndex] = useState(0)
  const [nextToken, setNextToken] = useState<string | undefined>(undefined)
  const [isRefreshDisabled, setIsRefreshDisabled] = useState(false)

  const fetchNodes = useCallback(async () => {
    if (!isInitialized) return

    try {
      const response = await Orchestrator.listNodes({
        query: {
          limit: 10000, // TODO: Remove this once pagination is implemented
          // limit: pageSize,
          next_token: pageIndex === 0 ? undefined : nextToken,
        },
        throwOnError: true,
      })

      const allNodes = response.data.Nodes ?? []
      setNodes(allNodes)
      const connected = allNodes.filter(
        (node) => getNodeConnectionStatus(node) == 'CONNECTED'
      ).length
      setConnectedNodes(connected)
      setNextToken(response.data.NextToken)
    } catch (error) {
      console.error('Error fetching nodes:', error)
      setNodes([])
      setConnectedNodes(0)
    }
  }, [isInitialized, pageIndex, nextToken])

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
    (node) =>
      node.Info?.NodeID?.toLowerCase().includes(search.toLowerCase()) ?? false
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
        <Badge
          variant="secondary"
          className="text-sm px-3 py-1 flex items-center gap-2"
        >
          <Server className="h-4 w-4" />
          <span>{connectedNodes} Connected</span>
        </Badge>
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
