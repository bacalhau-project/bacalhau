'use client'

import React, { Suspense } from 'react'
import { useSearchParams } from 'next/navigation'
import { NodesOverview } from '@/components/nodes/list/NodesOverview'
import NodeDetails from '@/components/nodes/details/NodeDetails'

function NodesContent() {
  const searchParams = useSearchParams()
  const id = searchParams.get('id')

  if (id) {
    return <NodeDetails nodeId={id} />
  }

  return <NodesOverview />
}

export default function NodesPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <NodesContent />
    </Suspense>
  )
}