import React, { useEffect, useCallback } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { Orchestrator, models_NodeState } from '@/lib/api/generated'
import NodeInformation from './NodeInformation'
import NodeInspect from './NodeInspect'
import { useApiOperation } from '@/hooks/useApiOperation'
import { ErrorDisplay } from '@/components/ErrorDisplay'

const NodeDetails = ({ nodeId }: { nodeId: string }) => {
  const {
    data: nodeData,
    isLoading,
    error,
    execute,
  } = useApiOperation<models_NodeState>()

  const fetchNodeData = useCallback(() => {
    execute(() =>
      Orchestrator.getNode({
        path: { id: nodeId },
        throwOnError: true,
      }).then((response) => response.data.Node!)
    )
  }, [execute, nodeId])

  useEffect(() => {
    fetchNodeData()
  }, [fetchNodeData])

  if (isLoading) return <NodeDetailsSkeleton />
  if (error) return <ErrorDisplay error={error} />
  if (!nodeData) return

  return (
    <div className="container mx-auto p-4">
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">{nodeData.Info?.NodeID}</h1>
        {/*<NodeActions node={nodeData} onNodeUpdated={fetchNodeData} />*/}
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
