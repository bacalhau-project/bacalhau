import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import Moment from 'react-moment';
import { bacalhauAPI } from '../../services/bacalhau';
import { Job, Execution } from '../../helpers/jobInterfaces';
import styles from './JobDetail.module.scss';
import { Layout } from '../../layout/Layout';
import {
  getShortenedJobID,
  fromTimestamp,
  capitalizeFirstLetter,
} from '../../helpers/helperFunctions';
import Container from '../../components/Container/Container';
import { ActionButton } from '../../components/ActionButton/ActionButton';
import Table from '../../components/Table/Table';
import JobInfo from './JobInfo/JobInfo';
import CliView from './CliView/CliView';

export const JobDetail: React.FC = () => {
  const { jobId } = useParams<{ jobId?: string }>();
  const [jobData, setJobData] = useState<Job | null>(null);
  const [jobExData, setJobExData] = useState<Execution[] | null>(null);
  const [selectedExecution, setSelectedExecution] = useState<
    Execution | undefined
  >(undefined);

  useEffect(() => {
    async function fetchData() {
      if (!jobId) return;

      try {
        const [jobResponse, executionsResponse] = await Promise.all([
          bacalhauAPI.describeJob(jobId),
          bacalhauAPI.jobExecution(jobId),
        ]);

        setJobData(jobResponse.Job);
        setJobExData(executionsResponse.Executions);
        setSelectedExecution(executionsResponse.Executions?.[0]);
      } catch (error) {
        console.error('Failed to fetch job data:', error);
      }
    }

    fetchData();
  }, [jobId]);

  if (!jobData || !jobExData) {
    return <div className={styles.loading}>Loading...</div>;
  }

  const manyExecutions = jobExData.length > 1;
  const executionsData = {
    headers: [
      'Execution ID',
      'Created',
      'Modified',
      'Node ID',
      'Status',
      'Action',
    ],
    rows: jobExData.map((item) => ({
      'Execution ID': item.ID,
      Created: (
        <Moment fromNow withTitle>
          {fromTimestamp(item.CreateTime)}
        </Moment>
      ),
      Modified: (
        <Moment fromNow withTitle>
          {fromTimestamp(item.ModifyTime)}
        </Moment>
      ),
      'Node ID': item.NodeID.slice(0, 7),
      Status: capitalizeFirstLetter(item.DesiredState.Message),
      Action: (
        <ActionButton text="Show" onClick={() => setSelectedExecution(item)} />
      ),
    })),
  };

  // TODO: figure out usful Inputs and Outputs data to display
  // const outData = {
  //   headers: ["Node Outputs"],
  //   rows: jobData.Tasks.flatMap((item) =>
  //     item.ResultPaths.map((path) => ({
  //       "Node Outputs": path.Name,
  //     })),
  //   ),
  // };

  return (
    <Layout pageTitle={`Job Detail | ${getShortenedJobID(jobData.ID)}`}>
      <div className={styles.jobDetail}>
        <div>
          <Container title="Job Overview">
            <JobInfo
              job={jobData}
              execution={selectedExecution}
              section="overview"
            />
            {manyExecutions && (
              <Table data={executionsData} style={{ fontSize: '12px' }} />
            )}
          </Container>
        </div>
        <div>
          <Container title="Execution Record">
            <JobInfo
              job={jobData}
              execution={selectedExecution}
              section="executionRecord"
            />
          </Container>
          <Container title="Standard Output">
            <CliView data={selectedExecution?.RunOutput.Stdout} />
          </Container>
          <Container title="Standard Error">
            <CliView data={selectedExecution?.RunOutput.stderr} />
          </Container>
          {/* <Container title={"Inputs"}>
            <Table data={inData} style={{ fontSize: "12px" }} />
          </Container> */}
          {/* <Container title={"Outputs"}>
            <Table data={outData} style={{ fontSize: "12px" }} />
          </Container> */}
        </div>
      </div>
    </Layout>
  );
};
