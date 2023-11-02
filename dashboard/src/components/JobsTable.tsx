// src/components/JobsTable.tsx
import React from "react";
import ProgramSummary from "./ProgramSummary";
import styles from "../../styles/JobsTable.module.scss";
import { Job, EngineSpec } from "../interfaces";

interface TableProps {
  data: Job[];
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

function parseData(jobs: Job[]): ParsedJobData[] {
  return jobs.map((job) => {
    const { Metadata, Spec } = job.Job;
    const shortenedJobID = job.Job.Metadata.ID.split("-")[0];
    return {
      id: shortenedJobID,
      name: shortenedJobID,
      createdAt: new Date(Metadata.CreatedAt).toLocaleString(),
      engineSpec: Spec.EngineSpec,
      jobType: "Batch",
      label: "Labels",
      status: "Status",
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
              <td>
                <ProgramSummary data={jobData.engineSpec} />
              </td>
              <td>{jobData.jobType}</td>
              <td>{jobData.label}</td>
              <td>{jobData.status}</td>
              <td>{jobData.action}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default JobsTable;
