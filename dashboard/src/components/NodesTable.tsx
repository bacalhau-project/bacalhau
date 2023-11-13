// src/components/NodesTable.tsx
import React from "react";
import styles from "../../styles/NodesTable.module.scss";
import Label from "./Label";
import ActionButton from "./ActionButton";
import {
  capitalizeFirstLetter,
  formatTimestamp,
} from "../helpers/helperFunctions";
import { Job, ParsedJobData } from "../helpers/interfaces";

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

// function parseData(nodes: Node[]): ParsedNodeData[] {
//   return nodes.map((node) => {

//   });
// }

const JobsTable: React.FC<TableProps> = ({ data }) => {
  // const parsedData = parseData(data);
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
          {/* {parsedData.map((jobData, index) => (
            <tr key={index}>
              <td className={styles.id}>{jobData.id}</td>
              <td className={styles.name}>{jobData.name}</td>
              <td className={styles.dateCreated}>{jobData.createdAt}</td>
              <td className={styles.program}>
                <ProgramSummary data={jobData.tasks} />
              </td>
              <td className={styles.jobType}>{jobData.jobType}</td>
              <td className={styles.label}>{jobData.label}</td>
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
          ))} */}
        </tbody>
      </table>
    </div>
  );
};

export default JobsTable;
