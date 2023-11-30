import React from 'react';
import { Job } from '../../../helpers/jobInterfaces';
import { getShortenedJobID, fromTimestamp, capitalizeFirstLetter } from "../../../helpers/helperFunctions";


interface JobInfoProps {
  job: Job;
  section: 'overview' | 'executionRecord' | 'executionDetails';
}

interface DataItem {
    label: string;
    value: string | undefined;
}  

const JobInfo: React.FC<JobInfoProps> = ({ job, section }) => {
    let dataToDisplay: DataItem[] = [];

    switch (section) {
        case 'overview':
            dataToDisplay = [
                { label: 'Job ID', value: getShortenedJobID(job.ID) },
                { label: 'Job Type', value: capitalizeFirstLetter(job.Type) },
                { label: 'Created', value: fromTimestamp(job.CreateTime).toString() },
                { label: 'Modified', value: fromTimestamp(job.ModifyTime).toString() },
                { label: 'Status', value: job.State.StateType },
                { label: 'Timeout Deadline', value: job.Tasks[0].Timeouts.ExecutionTimeout.toString() },
                { label: 'Executor Type', value: capitalizeFirstLetter(job.Tasks[0].Engine.Type) },
                { label: 'Image', value: job.Tasks[0].Engine.Params.Image },
                { label: 'GPU Details', value: job?.Tasks[0]?.Resources?.GPU ? job?.Tasks[0]?.Resources?.GPU : "Not specified" },
                { label: 'Timeout', value: job?.Tasks[0]?.Timeouts.ExecutionTimeout.toString() },
                { label: 'Requestor Node', value: 'Some value' },
                { label: 'Concurrency', value: 'Some value' }
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

    return (
        <div>
        {dataToDisplay.map(item => (
            <p key={item.label}>{item.label}: {item.value}</p>
        ))}
        </div>
    );
};

export default JobInfo;
