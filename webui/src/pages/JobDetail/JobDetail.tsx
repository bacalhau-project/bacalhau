import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { bacalhauAPI } from "../../services/bacalhau";
import { Job, Execution } from "../../helpers/jobInterfaces";
import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";
import { getShortenedJobID, fromTimestamp, capitalizeFirstLetter } from "../../helpers/helperFunctions";
import Container from "../../components/Container/Container";
import Table from '../../components/Table/Table';
import JobInfo from './JobInfo/JobInfo';

const JobDetail: React.FC = () => {
  const { jobId } = useParams<{ jobId?: string }>();
  const pageTitle = `Job Detail | ${jobId}`;

  const [jobData, setJobData] = useState<Job | null>(null);
  const [jobExData, setjobExData] = useState<Execution[] | null>(null);

  async function getJobData() {
    if (jobId) {
      try {
        const response = await bacalhauAPI.describeJob(jobId);
        if (response.Job) {
          setJobData(response.Job);
        }
      } catch (error) {
        console.error('Failed to fetch job data:', error);
      }
    }
  }

  async function getJobExecutionsData() {
    if (jobId) {
      try {
        const response = await bacalhauAPI.jobExecution(jobId);
        if (response.Executions) {
          setjobExData(response.Executions);
        }
      } catch (error) {
        console.error('Failed to fetch job data:', error);
      }
    }
  }

  useEffect(() => {
    getJobData();
    getJobExecutionsData();
  }, []);

  if (!jobData || !jobExData) {
    return <div>Loading...</div>;
  }

  console.log("jobData", jobData)
  console.log("jobExData", jobExData)

  getShortenedJobID(jobData.ID)

  const manyExecutions = jobExData.length > 1;

  const tableData = {
    headers: ["ID", "Created", "Modified", "Node ID", "Status", "Action"],
    rows: jobExData.map(item => ({
      "ID": item.ID,
      "Created": fromTimestamp(item.CreateTime).toString(),
      "Modified": fromTimestamp(item.ModifyTime).toString(),
      "Node ID": item.NodeID,
      "Status": capitalizeFirstLetter(item.DesiredState.Message),
      "Action": <button onClick={() => handleShowClick(item)}>Show</button>
    }))
  };
  
  const handleShowClick = (item: any) => {
    // Implement your logic here. For example, navigating to a detail page or showing more info
    console.log('Showing details for:', item.ID);
  };

  return (
    <Layout pageTitle={pageTitle}>
      <div className={styles.jobDetail}>
        <div>
          <Container title={"Job Overview"}>
            <JobInfo job={jobData} section="overview"/>
            {manyExecutions && (
              <Table data={tableData} style={{ fontSize: '12px' }}></Table>
            )}
          </Container>
        </div>
        <div>
          <Container title={"Execution Record"}>
            <JobInfo job={jobData} section="executionRecord"/>
          </Container>
          <Container title={"Standard Output"}>
            {/* <CliView data={} /> */}
          </Container>
          <Container title={"Standard Error"}>
            {/* <CliView data={} /> */}
          </Container>
          <Container title={"Inputs"}/>
          <Container title={"Outputs"}/>
        </div>
      </div>
    </Layout>
  );
};

export default JobDetail;
