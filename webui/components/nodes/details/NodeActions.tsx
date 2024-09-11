import React from 'react'
import { models_NodeState, OrchestratorService } from '@/lib/api/generated'
import { Button } from '@/components/ui/button'
import { useToast } from "@/hooks/use-toast"

// TODO: Implement the NodeActions component
interface NodeActionsProps {
  node: models_NodeState
  onNodeUpdated: () => void
}

const NodeActions: React.FC<NodeActionsProps> = ({ node, onNodeUpdated }) => {
  const { toast } = useToast()
  const handleApprove = async () => {
    try {
      await OrchestratorService.orchestratorUpdateNode(node.Info?.NodeID ?? '', {
        Action: 'approve',
        NodeID: node.Info?.NodeID,
      })
      toast({
        title: 'Node Approved',
        description: `Node ${node.Info?.NodeID} has been approved.`,
      })
      onNodeUpdated()
    } catch (error) {
      console.error('Error approving node:', error)
      toast({
        title: 'Error',
        description: 'Failed to approve the node. Please try again.',
        variant: 'destructive',
      })
    }
  }

  const handleReject = async () => {
    try {
      await OrchestratorService.orchestratorUpdateNode(node.Info?.NodeID ?? '', {
        Action: 'reject',
        NodeID: node.Info?.NodeID,
      })
      toast({
        title: 'Node Rejected',
        description: `Node ${node.Info?.NodeID} has been rejected.`,
      })
      onNodeUpdated()
    } catch (error) {
      console.error('Error rejecting node:', error)
      toast({
        title: 'Error',
        description: 'Failed to reject the node. Please try again.',
        variant: 'destructive',
      })
    }
  }

  return (
    <div className="space-x-2">
      {node.Membership?.membership !== 2 && (
        <Button onClick={handleApprove}>Approve</Button>
      )}
      {node.Membership?.membership !== 3 && (
        <Button onClick={handleReject} variant="destructive">
          Reject
        </Button>
      )}
    </div>
  )
}

export default NodeActions