// src/pages/JobsDashboard.tsx

import React from 'react'
import Table from '../components/Table'

const JobsDashboard: React.FC = () => {
  const headers = [
    'Job ID',
    'Name',
    'Created',
    'Program',
    'Job Type',
    'Label',
    'Status',
    'Action',
  ]
  const data = [
    [
      'xxxxxxxx',
      'Long Running Job #1',
      '2 minutes ago',
      'Ubuntu',
      'Daemon',
      'Canary',
      'Running',
      'View',
    ],
    [
      'xxxxxxxx',
      '',
      'October 20, 2023',
      'Ubuntu',
      'Batch',
      '',
      'Complete',
      'View',
    ],
    // ... other rows
  ]

  return (
    <div>
      <h1>Jobs Dashboard</h1>
      <Table headers={headers} data={data} />
    </div>
  )
}

export default JobsDashboard
