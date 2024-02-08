import React, { useEffect, useState } from "react"
import styles from "./JobsDashboard.module.scss"
import { JobsTable } from "./JobsTable/JobsTable"
import { Layout } from "../../layout/Layout"
import { Job } from "../../helpers/jobInterfaces"
import { bacalhauAPI } from "../../services/bacalhau"
import { TableSettingsContextProvider } from "../../context/TableSettingsContext"

interface JobsDashboardProps {
  pageTitle?: string
}

export const JobsDashboard: React.FC<JobsDashboardProps> = ({
  pageTitle = "Jobs Dashboard",
}) => {
  const [data, setData] = useState<Job[]>([])

  useEffect(() => {
    try {
      bacalhauAPI
        .listJobs()
        .then((response) => response.Jobs)
        .then((jobs) => {
          if (jobs) {
            setData(jobs)
          }
        })
    } catch (error) {
      console.error(error)
    }
  }, [])

  return (
    <Layout pageTitle={pageTitle}>
      <div className={styles.jobsDashboard} data-testid="jobsTableContainer">
        <TableSettingsContextProvider>
          <JobsTable data={data} />
        </TableSettingsContextProvider>
      </div>
    </Layout>
  )
}

JobsDashboard.defaultProps = {
  pageTitle: "Jobs Dashboard",
}
