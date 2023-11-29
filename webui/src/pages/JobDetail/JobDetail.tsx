import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { bacalhauAPI } from "../../services/bacalhau";
import { Job, Execution } from "../../helpers/jobInterfaces";
import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";
import { getShortenedJobID } from "../../helpers/helperFunctions";
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

  const tableHeaders = ["ID", "Created", "Modified", "Node ID", "Status", "Action"];
  // TEMP
  const tableData = {
    rows: [
      { ID: 1, "Node ID": "Node 1", Status: "Active" },
      { ID: 2, "Node ID": "Node 2", Status: "Inactive" },
    ]
  };

  return (
    <Layout pageTitle={pageTitle}>
      <div className={styles.jobDetail}>
        <div>
          <Container title={"Job Overview"}>
            <JobInfo job={jobData} section="overview"/>
          </Container>
          <Container title={"Execution Record"}>
            <JobInfo job={jobData} section="executionRecord"/>
          </Container>
        </div>
        <div>
          <Container title={"Execution Details"}>
            <JobInfo job={jobData} section="executionDetails"/>
            <Table headers={tableHeaders} data={tableData} style={{ fontSize: '12px' }}></Table>
          </Container>
          <Container title={"Standard Output"}>
            {/* <CliView data={} /> */}
          </Container>
          <Container title={"Execution Logs"}/>
        </div>
        <div>
          <Container title={"Inputs"}/>
          <Container title={"Input"}/>
          <Container title={"Outputs"}/>
          <Container title={"Output"}/>
        </div>
      </div>
    </Layout>
  );
};

export default JobDetail;
