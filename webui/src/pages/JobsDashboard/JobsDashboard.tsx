import React, { useEffect, useState } from "react"
import styles from "./JobsDashboard.module.scss"
import { JobsTable } from "./JobsTable/JobsTable"
import { Layout } from "../../layout/Layout"
import { Job } from "../../helpers/jobInterfaces"
import { bacalhauAPI } from "../../services/bacalhau"

export const JobsDashboard: React.FC = () => {
  const [data, setData] = useState<Job[]>([])

  async function getJobsData() {
    try {
      const response = await bacalhauAPI.listJobs()
      if (response.Jobs) {
        setData(response.Jobs)
      }
    } catch (error) {
      console.error(error)
    }
  }

  useEffect(() => {
    getJobsData()
  }, [])

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsDashboard}>
        <JobsTable data={data} />
      </div>
    </Layout>
  )
}
