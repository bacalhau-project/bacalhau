import React, { useState, useEffect, useCallback } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { OrchestratorService, models_NodeState } from '@/lib/api/generated'
import { useApi } from '@/app/providers/ApiProvider'
import NodeInformation from './NodeInformation'
import NodeInspect from './NodeInspect'
import NodeActions from './NodeActions'

const NodeDetails = ({ nodeId }: { nodeId: string }) => {
  const [nodeData, setNodeData] = useState<models_NodeState | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { isInitialized } = useApi()

  const fetchNodeData = useCallback(async () => {
    if (!isInitialized) return

    setIsLoading(true)
    setError(null)
    try {
      const response = await OrchestratorService.orchestratorGetNode(nodeId)
      setNodeData(response.Node!)
    } catch (error) {
      console.error('Error fetching node data:', error)
      setError('Failed to fetch node data. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }, [isInitialized, nodeId])

  useEffect(() => {
    fetchNodeData()
  }, [fetchNodeData])

  const handleNodeUpdated = () => {
    fetchNodeData()
  }

  if (isLoading) return <NodeDetailsSkeleton />
  if (error) return <Alert variant="destructive">{error}</Alert>
  if (!nodeData) return <Alert>Node not found.</Alert>

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{nodeData.Info?.NodeID}</h1>
        {/*<NodeActions node={nodeData} onNodeUpdated={handleNodeUpdated} />*/}
      </div>

      <Tabs defaultValue="information">
        <TabsList>
          <TabsTrigger value="information">Information</TabsTrigger>
          <TabsTrigger value="inspect">Inspect</TabsTrigger>
        </TabsList>
        <TabsContent value="information">
          <NodeInformation node={nodeData} />
        </TabsContent>
        <TabsContent value="inspect">
          <NodeInspect node={nodeData} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

const NodeDetailsSkeleton = () => (
  <div className="container mx-auto p-4">
    <Skeleton className="h-8 w-64 mb-4" />
    <Skeleton className="h-[400px] w-full" />
  </div>
)

export default NodeDetails