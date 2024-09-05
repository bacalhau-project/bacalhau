'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { getJob } from '@/lib/api'

export function JobDetails({ id }: { id: string }) {
  const [job, setJob] = useState(null)

  useEffect(() => {
    const fetchJob = async () => {
      const data = await getJob(id)
      setJob(data.Job)
    }
    fetchJob()
  }, [id])

  if (!job) return <div>Loading...</div>

  return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Job Details: {job.Name}</h1>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <h2 className="text-xl font-semibold mb-2">Specification</h2>
            <pre className="bg-gray-100 p-4 rounded">{JSON.stringify(job, null, 2)}</pre>
          </div>
          <div>
            <h2 className="text-xl font-semibold mb-2">Executions</h2>
            <ul>
              {job.Executions.map((execution) => (
                  <li key={execution.ID}>
                    <Link href={`/executions/${execution.ID}`}>{execution.ID}</Link>
                  </li>
              ))}
            </ul>
            <h2 className="text-xl font-semibold mt-4 mb-2">History</h2>
            <ul>
              {job.History.map((event, index) => (
                  <li key={index}>{event.EventName}: {event.Status}</li>
              ))}
            </ul>
          </div>
        </div>
      </div>
  )
}