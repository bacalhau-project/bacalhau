// src/components/JobsTable.tsx
import React from "react";
import ProgramSummary from "./ProgramSummary";
import Label from "./Label";
import styles from "../../styles/JobsTable.module.scss";
import { Job, EngineSpec } from "../interfaces";
import ActionButton from "./ActionButton";

interface TableProps {
  data: Job[];
}

interface FlexibleJob {
  [key: string]: any;
}

interface ParsedJobData {
  id: string;
  name: string;
  createdAt: string;
  engineSpec: EngineSpec;
  jobType: string;
  label: string;
  status: string;
  action: string;
}

const labelColorMap: { [key: string]: string } = {
  running: "green",
  warning: "orange",
  error: "red",
  paused: "blue",
  stopped: "grey",
  complete: "green",
  progress: "orange",
  failed: "red"
};

function parseData(jobs: Job[]): ParsedJobData[] {
  const status = "Complete" // TODO: hardcoded as not yet available

  return jobs.map((job) => {
    const { Metadata, Spec } = job.Job;
    const shortenedJobID = job.Job.Metadata.ID.split("-")[0];
    const jobType = (job.Job as FlexibleJob).jobType ?? "Batch"; // TODO: link up `Job Type` when inplementing new api version
    return {
      id: shortenedJobID,
      name: shortenedJobID,
      createdAt: new Date(Metadata.CreatedAt).toLocaleString(),
      engineSpec: Spec.EngineSpec,
      jobType: jobType,
      label: "",
      status: status,
      action: "Action",
    };
  });
}

const JobsTable: React.FC<TableProps> = ({ data }) => {
  const parsedData = parseData(data);
  return (
    <div className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            <th className={styles.jobID}>Job ID</th>
            <th>Name</th>
            <th className={styles.dateCreated}>Created</th>
            <th>Program</th>
            <th>Job Type</th>
            <th>Label</th>
            <th>Status</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {parsedData.map((jobData, index) => (
            <tr key={index}>
              <td>{jobData.id}</td>
              <td>{jobData.name}</td>
              <td className={styles.dateCreated}>{jobData.createdAt}</td>
              <td className={styles.program}>
                <ProgramSummary data={jobData.engineSpec} />
              </td>
              <td>{jobData.jobType}</td>
              <td>{jobData.label}</td>
              <td className={styles.status}>
                <Label
                  text={jobData.status}
                  color={labelColorMap[jobData.status.toLowerCase()]}
                />
              </td>
              <td className={styles.action}>
                <ActionButton text="View" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default JobsTable;
