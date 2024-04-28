import React, { useEffect, useState } from "react"
import { toast } from "react-toastify"
import styles from "./JobsDashboard.module.scss"
import { JobsTable } from "./JobsTable/JobsTable"
import { Layout } from "../../layout/Layout"
import { Job } from "../../helpers/jobInterfaces"
import { bacalhauAPI } from "../../services/bacalhau"
import { TableSettingsContextProvider } from "../../context/TableSettingsContext"
import { AxiosError } from "axios"
import { useLocation, useNavigate } from "react-router-dom"

interface JobsDashboardProps {
  pageTitle?: string
}

export const JobsDashboard: React.FC<JobsDashboardProps> = ({
  pageTitle = "Jobs Dashboard",
}) => {
  const [data, setData] = useState<Job[]>([])
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    bacalhauAPI
      .listJobs()
      .then((response) => response.Jobs)
      .then((jobs) => {
        if (jobs) {
          setData(jobs)
        }
      }).catch(error => {
        if (error instanceof AxiosError) {
          switch (error.response?.status) {
            case 401:
              // User has a bad auth token
              toast("Your session has expired. Please authenticate again.", { type: "error", toastId: "session-expired" })
              navigate("/Auth", { state: { prev: location } })
              break
            case 403:
              toast("This page requires authentication. Please authenticate.", { type: "warning", toastId: "auth-required" })
              navigate("/Auth", { state: { prev: location } })
              break
            default:
              toast(error.response?.statusText, { type: "error" })
          }
        } else {
          console.error(error)
        }
      })
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