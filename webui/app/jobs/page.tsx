'use client'

import React, { Suspense } from 'react'
import { useSearchParams } from 'next/navigation'
import { JobsOverview } from '@/components/jobs/JobsOverview'
import JobDetails from '@/components/jobs/details/JobDetails'

function JobsContent() {
  const searchParams = useSearchParams()
  const id = searchParams.get('id')

  if (id) {
    return <JobDetails jobId={id} />
  }

  return <JobsOverview />
}

export default function JobsPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <JobsContent />
    </Suspense>
  )
}