import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { models_NodeState } from '@/lib/api/generated'
import {
  ConnectionStatus,
  MembershipStatus,
} from '@/components/nodes/NodeStatus'
import { NodeResources } from '@/components/nodes/NodeResources'
import Labels from '@/components/Labels'
import InfoItem from '@/components/InfoItem'

interface NodeInformationProps {
  node: models_NodeState
}

const NodeInformation: React.FC<NodeInformationProps> = ({ node }) => {
  const computeInfo = node.Info?.ComputeNodeInfo

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Card>
        <CardContent className="pt-6">
          <h2 className="text-xl font-bold mb-4">Basic Information</h2>
          <InfoItem label="Node ID">{node.Info?.NodeID}</InfoItem>
          <InfoItem label="Node Type">{node.Info?.NodeType}</InfoItem>
          <InfoItem label="Connection">
            <ConnectionStatus node={node} />
          </InfoItem>
          <InfoItem label="Membership">
            <MembershipStatus node={node} />
          </InfoItem>
          <InfoItem label="Version">
            {node.Info?.BacalhauVersion?.GitVersion || 'N/A'}
          </InfoItem>
          <InfoItem label="Running">{computeInfo?.RunningExecutions}</InfoItem>
          <InfoItem label="Enqueued">
            {computeInfo?.EnqueuedExecutions}
          </InfoItem>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <h2 className="text-xl font-bold mb-4">Resources</h2>
          <NodeResources node={node} variant="large" />
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <h2 className="text-xl font-bold mb-4">Capabilities</h2>
          <InfoItem label="Engines">
            <Labels
              labels={computeInfo?.ExecutionEngines}
              color="bg-blue-100 text-blue-800"
            />
          </InfoItem>
          <InfoItem label="Publishers">
            <Labels
              labels={computeInfo?.Publishers}
              color="bg-green-100 text-green-800"
            />
          </InfoItem>
          <InfoItem label="Storage">
            <Labels
              labels={computeInfo?.StorageSources}
              color="bg-purple-100 text-purple-800"
            />
          </InfoItem>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <h2 className="text-xl font-bold mb-4">Labels</h2>
          <Labels labels={node.Info?.Labels} />
        </CardContent>
      </Card>
    </div>
  )
}

export default NodeInformation
