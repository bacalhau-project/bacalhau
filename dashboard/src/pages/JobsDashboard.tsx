// src/pages/JobsDashboard.tsx

import React from 'react'
import styles from '../../styles/JobsDashboard.module.scss';
import Table from '../components/Table'
import Layout from '../components/Layout';

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
    // ... test data
  ]

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsdashboard}>
        <h1>Jobs Dashboard</h1>
        <Table headers={headers} data={data} />
      </div>
    </Layout>
  )
}

export default JobsDashboard
