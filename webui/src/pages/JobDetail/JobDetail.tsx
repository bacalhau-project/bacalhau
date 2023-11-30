import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { bacalhauAPI } from "../../services/bacalhau";
import { Job, Execution } from "../../helpers/jobInterfaces";
import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";
import { getShortenedJobID } from "../../helpers/helperFunctions";
import Container from "../../components/Container/Container";
import JobInfo from './JobInfo/JobInfo';

const JobDetail: React.FC = () => {
  const { jobId } = useParams<{ jobId?: string }>();
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

  const pageTitle = `Job Detail | ${ getShortenedJobID(jobData.ID)}`;

  return (
    <Layout pageTitle={pageTitle}>
      <div className={styles.jobDetail}>
        <div>
          <Container title={"Job Overview"}>
            <JobInfo job={jobData} executions={jobExData} section="overview"/>
          </Container>
        </div>
        <div>
          <Container title={"Execution Record"}>
            <JobInfo job={jobData} executions={jobExData} section="executionRecord"/>
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
