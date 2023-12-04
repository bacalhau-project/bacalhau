import React, { useState, useEffect } from 'react';
import { Execution, Job } from '../../../helpers/jobInterfaces';
import { fromTimestamp, capitalizeFirstLetter } from "../../../helpers/helperFunctions";
import styles from "./JobInfo.module.scss";

interface JobInfoProps {
    job: Job;
    execution: Execution | undefined;
    section: 'overview' | 'executionRecord' | 'stdout' | 'stderr' | 'inputs' | 'outputs';
}

interface DataItem {
    label: string;
    value: string | undefined;
}  

const JobInfo: React.FC<JobInfoProps> = ({ job, execution, section }) => {
    const [dataToDisplay, setDataToDisplay] = useState<DataItem[]>([]);

    useEffect(() => {
        switch (section) {
            case 'overview':
                setDataToDisplay([
                    { label: 'Job ID', value: job.ID },
                    { label: 'Job Type', value: capitalizeFirstLetter(job.Type) },
                    { label: 'Created', value: fromTimestamp(job.CreateTime).toString() },
                    { label: 'Modified', value: fromTimestamp(job.ModifyTime).toString() },
                    { label: 'Status', value: job.State.StateType },
                    { label: 'Timeout Deadline', value: job.Tasks[0].Timeouts.ExecutionTimeout.toString() },
                    { label: 'Executor Type', value: capitalizeFirstLetter(job.Tasks[0].Engine.Type) },
                    { label: 'Image', value: job.Tasks[0].Engine.Params.Image },
                    { label: 'GPU Details', value: job?.Tasks[0]?.Resources?.GPU ? job?.Tasks[0]?.Resources?.GPU : "Not specified" },
                    { label: 'Timeout', value: job?.Tasks[0]?.Timeouts.ExecutionTimeout.toString() },
                    { label: 'Requestor Node', value: job.Meta["bacalhau.org/requester.id"]},
                    // { label: 'Concurrency', value: '' }
                ]);
                break;
            case 'executionRecord':
                if (execution !== undefined)
                setDataToDisplay([
                    { label: 'Execution ID', value: execution.ID },
                    { label: 'Initiation Time', value: fromTimestamp(execution.CreateTime).toString() },
                    { label: 'Completion Time', value: fromTimestamp(execution.ModifyTime).toString() },
                    { label: 'Exit Code', value: execution.RunOutput.exitCode?.toString() },
                    { label: 'Execution Note', value: capitalizeFirstLetter(execution.DesiredState.Message) }
                ]);
                break;
            case 'stdout':
                if (execution !== undefined)
                setDataToDisplay([
                    { label: 'Initiation Time', value: fromTimestamp(execution.CreateTime).toString() },
                ]);
                break;
            case 'stderr':
                if (execution !== undefined)
                setDataToDisplay([
                    { label: 'Initiation Time', value: fromTimestamp(execution.CreateTime).toString() },
                ]);
                break;
            case 'inputs':
                if (execution !== undefined)
                setDataToDisplay([
                    { label: 'Initiation Time', value: fromTimestamp(execution.CreateTime).toString() },
                ]);
                break;
            case 'outputs':
                if (execution !== undefined)
                setDataToDisplay([
                    { label: 'Initiation Time', value: fromTimestamp(execution.CreateTime).toString() },
                ]);
                break;
            default:
                break;
        }
    }, [section, execution]);

    return (
        <div className={styles.jobInfo}>
            {dataToDisplay.map(item => (
                <p key={item.label} className={styles.item}>
                    <span className={styles.key}>{item.label}: </span>
                    <span className={styles.value}>{item.value}</span> 
                </p>
            ))}
        </div>
    );
};

export default JobInfo;
