import React, { useState, useEffect } from "react";
import Moment from "react-moment";
import { Execution, Job } from "../../../helpers/jobInterfaces";
import {
  fromTimestamp,
  capitalizeFirstLetter,
} from "../../../helpers/helperFunctions";
import styles from "./JobInfo.module.scss";

interface JobInfoProps {
  job: Job;
  execution: Execution | undefined;
  section:
    | "overview"
    | "executionRecord"
    | "stdout"
    | "stderr"
    | "inputs"
    | "outputs";
}

interface DataItem {
  label: string;
  value: string | JSX.Element | undefined;
}

const JobInfo: React.FC<JobInfoProps> = ({ job, execution, section }) => {
  const [dataToDisplay, setDataToDisplay] = useState<DataItem[]>([]);

  useEffect(() => {
    switch (section) {
      case "overview":
        setDataToDisplay([
          { label: "Job ID", value: job.ID },
          { label: "Job Type", value: capitalizeFirstLetter(job.Type) },
          { label: "Created", value: fromTimestamp(job.CreateTime).toString() },
          {
            label: "Modified",
            value: fromTimestamp(job.ModifyTime).toString(),
          },
          { label: "Status", value: job.State.StateType },
          {
            label: "Timeout Deadline",
            value: job.Tasks[0].Timeouts.ExecutionTimeout.toString(),
          },
          {
            label: "Executor Type",
            value: capitalizeFirstLetter(job.Tasks[0].Engine.Type),
          },
          { label: "Image", value: job.Tasks[0].Engine.Params.Image },
          {
            label: "GPU Details",
            value: job?.Tasks[0]?.Resources?.GPU
              ? job?.Tasks[0]?.Resources?.GPU
              : "Not specified",
          },
          {
            label: "Timeout",
            value: job?.Tasks[0]?.Timeouts.ExecutionTimeout.toString(),
          },
          {
            label: "Requestor Node",
            value: job.Meta["bacalhau.org/requester.id"],
          },
          // { label: 'Concurrency', value: '' }
        ]);
        break;
      case "executionRecord":
        if (execution !== undefined)
          setDataToDisplay([
            { label: "Execution ID", value: execution.ID },
            {
              label: "Initiation Time",
              value: (
                <Moment fromNow withTitle>
                  {fromTimestamp(execution.CreateTime)}
                </Moment>
              ),
            },
            {
              label: "Completion Time",
              value: (
                <Moment fromNow withTitle>
                  {fromTimestamp(execution.ModifyTime)}
                </Moment>
              ),
            },
            {
              label: "Exit Code",
              value: execution.RunOutput.exitCode?.toString(),
            },
            {
              label: "Execution Note",
              value: capitalizeFirstLetter(execution.DesiredState.Message),
            },
          ]);
        break;
      default:
        break;
    }
  }, [section, execution, job]);

  return (
    <div className={styles.jobInfo}>
      {dataToDisplay.map((item) => (
        <p key={item.label} className={styles.item}>
          <span className={styles.key}>{item.label}: </span>
          <span className={styles.value}>{item.value}</span>
        </p>
      ))}
    </div>
  );
};

export default JobInfo;
