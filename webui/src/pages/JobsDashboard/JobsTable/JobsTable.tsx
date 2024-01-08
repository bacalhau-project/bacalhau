import React, { useContext } from "react";
import Moment from "react-moment";
import styles from "./JobsTable.module.scss";
import ProgramSummary from "./ProgramSummary/ProgramSummary";
import Label from "../../../components/Label/Label";
import { ActionButton } from "../../../components/ActionButton/ActionButton";
import {
  capitalizeFirstLetter,
  fromTimestamp,
  getShortenedJobID,
  createLabelArray,
} from "../../../helpers/helperFunctions";
import { Job, ParsedJobData } from "../../../helpers/jobInterfaces";
import TableSettingsContext from "../../../context/TableSettingsContext";

interface TableProps {
  data: Job[];
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
};

function parseData(jobs: Job[]): ParsedJobData[] {
  return jobs.map((job) => {
    if (!job.Tasks || job.Tasks.length === 0) {
      throw new Error(`Job with ID: ${job.ID} has no tasks.`);
    }
    const firstTask = job.Tasks[0];
    const jobType = job.Type ?? "batch";
    const jobShortID = getShortenedJobID(job.ID);
    const jobName = job.Name;

    if (jobType === "batch") {
      job.Name = jobShortID;
    } else {
      job.Name = jobName;
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
    };
  });
}

export const JobsTable: React.FC<TableProps> = ({ data }) => {
  const { settings } = useContext(TableSettingsContext);
  const parsedData = parseData(data);

  return (
    <div className={styles.tableContainer}>
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
          {parsedData.map((jobData, index) => (
            <tr key={index}>
              {settings.showJobName && (
                <td className={styles.name}>{jobData.name}</td>
              )}
              {settings.showCreated && (
                <td className={styles.dateCreated}>
                  <Moment fromNow withTitle>
                    {jobData.createdAt}
                  </Moment>
                </td>
              )}
              {settings.showProgram && (
                <td className={styles.program}>
                  <ProgramSummary data={jobData.tasks} />
                </td>
              )}
              {settings.showJobType && (
                <td className={styles.jobType}>{jobData.jobType}</td>
              )}
              {settings.showLabel && (
                <td className={styles.label}>
                  {jobData.label.map((label) => (
                    <Label text={label} color={"grey"} />
                  ))}
                </td>
              )}
              {settings.showStatus && (
                <td className={styles.status}>
                  <Label
                    text={jobData.status}
                    color={labelColorMap[jobData.status.toLowerCase()]}
                  />
                </td>
              )}
              {settings.showAction && (
                <td className={styles.action}>
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
  );
};
