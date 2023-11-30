import React from 'react';
import { Execution, Job } from '../../../helpers/jobInterfaces';
import { fromTimestamp, capitalizeFirstLetter } from "../../../helpers/helperFunctions";
import styles from "./JobInfo.module.scss";
import Table from '../../../components/Table/Table';

interface JobInfoProps {
  job: Job;
  executions: Execution[];
  section: 'overview' | 'executionRecord' | 'executionDetails';
}

interface DataItem {
    label: string;
    value: string | undefined;
}  

const JobInfo: React.FC<JobInfoProps> = ({ job, executions, section }) => {
    let dataToDisplay: DataItem[] = [];

    switch (section) {
        case 'overview':
            dataToDisplay = [
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
                { label: 'Concurrency', value: '' }
            ];
            break;
        case 'executionRecord':
            dataToDisplay = [
                { label: 'Initiation Time', value: 'Some value' },
                { label: 'Completion Time', value: 'Some value' },
                { label: 'Exit Code', value: 'Some value' },
                { label: 'Standard Error', value: 'Some value' },
                { label: 'Execution Note', value: 'Some value' }
            ];
            break;
        case 'executionDetails':
            dataToDisplay = [

            ];
            break;
        default:
            break;
    }

    const manyExecutions = executions.length > 1;
    const tableData = {
        headers: ["ID", "Created", "Modified", "Node ID", "Status", "Action"],
        rows: executions.map(item => ({
            "ID": item.ID,
            "Created": fromTimestamp(item.CreateTime).toString(),
            "Modified": fromTimestamp(item.ModifyTime).toString(),
            "Node ID": item.NodeID,
            "Status": capitalizeFirstLetter(item.DesiredState.Message),
            "Action": <button onClick={() => handleShowClick(item)}>Show</button>
        }))
    };

    const handleShowClick = (item: any) => {
        // TODO: Logic to show selected execution
        console.log('Showing details for:', item.ID);
    };

    return (
        <div className={styles.jobInfo}>
            {dataToDisplay.map(item => (
                <p key={item.label} className={styles.item}>
                    <span className={styles.key}>{item.label}: </span>
                    <span className={styles.value}>{item.value}</span> 
                </p>
            ))}
            {manyExecutions && section=='overview' && (
              <div>
                <span className={styles.key}>Executions List:</span>
                    <Table data={tableData} style={{ fontSize: '12px' }}></Table>
              </div>
            )}
        </div>
    );
};

export default JobInfo;
