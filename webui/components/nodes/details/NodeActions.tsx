import React from 'react'
import { models_NodeState, Orchestrator } from '@/lib/api/generated'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/use-toast'

// TODO: Implement the NodeActions component
interface NodeActionsProps {
  node: models_NodeState
  onNodeUpdated: () => void
}

const NodeActions: React.FC<NodeActionsProps> = ({ node, onNodeUpdated }) => {
  const { toast } = useToast()

  const handleUpdateNode = async (action: 'approve' | 'reject') => {
    try {
      await Orchestrator.updateNode({
        path: { id: node.Info?.NodeID ?? '' },
        body: {
          Action: action,
          NodeID: node.Info?.NodeID,
        },
        throwOnError: true,
      })
      toast({
        title: `Node ${action === 'approve' ? 'Approved' : 'Rejected'}`,
        description: `Node ${node.Info?.NodeID} has been ${action === 'approve' ? 'approved' : 'rejected'}.`,
      })
      onNodeUpdated()
    } catch (error) {
      console.error(`Error ${action}ing node:`, error)
      toast({
        title: 'Error',
        description: `Failed to ${action} the node. Please try again.`,
        variant: 'destructive',
      })
    }
  }

  const handleApprove = () => handleUpdateNode('approve')
  const handleReject = () => handleUpdateNode('reject')

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
