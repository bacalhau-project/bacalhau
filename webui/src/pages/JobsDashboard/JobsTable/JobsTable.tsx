import React, { useContext } from "react"
import { formatDistanceToNow } from "date-fns"
import styles from "./JobsTable.module.scss"
import ProgramSummary from "./ProgramSummary/ProgramSummary"
import Label from "../../../components/Label/Label"
import { ActionButton } from "../../../components/ActionButton/ActionButton"
import {
  capitalizeFirstLetter,
  fromTimestamp,
  getShortenedJobID,
  createLabelArray,
} from "../../../helpers/helperFunctions"
import { Job, ParsedJobData } from "../../../helpers/jobInterfaces"
import TableSettingsContext from "../../../context/TableSettingsContext"
import { Task } from "../../../models/task"

interface TableProps {
  data: Job[]
}

const labelColorMap: { [key: string]: string } = {
  running: "green",
  warning: "orange",
  error: "red",
  paused: "blue",
  stopped: "grey",
  complete: "green",
  progress: "orange",
  failed: "red",
}

function parseData(jobs: Job[]): ParsedJobData[] {
  if (!jobs) {
    console.log("No jobs data provided.")
    return []
  }

  const ParsedJobDataReturn = jobs.map((job) => {
    if (! job.Tasks || job.Tasks.length === 0) {
      throw new Error(`Job with ID: ${job.ID} has no tasks.`)
    }
    const firstTask = (job.Tasks && job.Tasks[0]) ?? new Task("--")
    const jobType = job.Type ?? "batch"
    const jobShortID = getShortenedJobID(job.ID)
    const jobName = job.Name

    if (jobType === "batch" && jobName === "") {
      job.Name = jobShortID
    } else {
      job.Name = jobName
    }
    return {
      longId: job.ID,
      name: job.Name,
      createdAt: fromTimestamp(job.CreateTime),
      tasks: firstTask,
      jobType: capitalizeFirstLetter(jobType),
      label: createLabelArray(job.Labels),
      status: job.State.StateType,
      action: "Action",
    }
  })
  return ParsedJobDataReturn
}

export const JobsTable: React.FC<TableProps> = ({ data }) => {
  const { settings } = useContext(TableSettingsContext)

  const parsedData = parseData(data)

  return (
    <div id="jobsTableContainer" className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            {settings.showJobName && <th className={styles.jobName}>Job</th>}
            {settings.showCreated && (
              <th className={styles.dateCreated}>Created</th>
            )}
            {settings.showProgram && <th>Program</th>}
            {settings.showJobType && <th>Job Type</th>}
            {settings.showLabel && <th>Label</th>}
            {settings.showStatus && <th>Status</th>}
          </tr>
        </thead>
        <tbody>
          {parsedData.map((jobData, _index) => (
            <tr key={jobData.longId} data-testid="jobRow">
              {settings.showJobName && (
                <td className={styles.name}>{jobData.name}</td>
              )}
              {settings.showCreated && (
                <td className={styles.dateCreated}>
                  {formatDistanceToNow(jobData.createdAt)}
                </td>
              )}
              {settings.showProgram && (
                <td className={styles.program} aria-label="tasks">
                  <ProgramSummary data={jobData.tasks} />
                </td>
              )}
              {settings.showJobType && (
                <td className={styles.jobType}>{jobData.jobType}</td>
              )}
              {settings.showLabel && (
                <td className={styles.label}>
                  {jobData.label.map((label) => (
                    // Render label key with job ID to avoid duplicate keys
                    <Label
                      text={label}
                      color="grey"
                      key={`label-${jobData.longId}-${label}`}
                    />
                  ))}
                </td>
              )}
              {settings.showStatus && (
                <td className={styles.status} aria-label="status">
                  <Label
                    text={jobData.status}
                    color={labelColorMap[jobData.status.toLowerCase()]}
                    key={`status-${jobData.longId}`}
                  />
                </td>
              )}
              {settings.showAction && (
                <td className={styles.action} aria-label="view details">
                  <ActionButton
                    text="View"
                    to="/JobDetail"
                    id={jobData.longId}
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
