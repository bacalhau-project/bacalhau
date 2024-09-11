import React from 'react'
import { Badge } from '@/components/ui/badge'
import { models_NodeState, models_NodeMembershipState } from '@/lib/api/generated'
import { getNodeConnectionStatus, getNodeMembershipStatus } from '@/lib/api/utils'

const getColorForStatus = (status: string): string => {
  const colors: Record<string, string> = {
    CONNECTED: 'bg-green-100 text-green-800',
    DISCONNECTED: 'bg-red-100 text-red-800',
    UNKNOWN: 'bg-gray-100 text-gray-800',
    PENDING: 'bg-yellow-100 text-yellow-800',
    APPROVED: 'bg-green-100 text-green-800',
    REJECTED: 'bg-red-100 text-red-800',
  }

  return colors[status] || 'bg-gray-100 text-gray-800'
}

// Connection Status Component
interface ConnectionStatusProps {
  node: models_NodeState
}

export const ConnectionStatus: React.FC<ConnectionStatusProps> = ({ node }) => {
  const status = getNodeConnectionStatus(node)
  const colorClass = getColorForStatus(status)

  return (
    <Badge className={`text-xs ${colorClass}`}>
      {status}
    </Badge>
  )
}

// Membership Status Component
interface MembershipStatusProps {
  node: models_NodeState
}

export const MembershipStatus: React.FC<MembershipStatusProps> = ({ node }) => {
  const status = getNodeMembershipStatus(node)
  const colorClass = getColorForStatus(status)

  return (
    <Badge className={`text-xs ${colorClass}`}>
      {status}
    </Badge>
  )
}