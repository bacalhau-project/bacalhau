import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";
import Container from "../../components/Container/Container";
import { bacalhauAPI } from "../../services/bacalhau";
import { Job } from "../../helpers/jobInterfaces";


const JobDetail: React.FC = () => {
  const { jobId } = useParams<{ jobId?: string }>();
  const [data, setData] = useState<Job | null>(null);

  async function getJobData() {
    if (jobId) {
      try {
        const response = await bacalhauAPI.describeJob(jobId);
        if (response.Job) {
          setData(response.Job);
        }
      } catch (error) {
        console.error('Failed to fetch job data:', error);
      }
    }
  }

  useEffect(() => {
    getJobData();
  }, []);

  if (!data) {
    return <div>Loading...</div>;
  }

  const pageTitle = `Job Detail | ${jobId}`;

  return (
    <Layout pageTitle={pageTitle}>
      <div className={styles.jobDetail}>
        <div>
          <Container title={"Job Overview"}/>
          <Container title={"Execution Record"}/>
        </div>
        <div>
          <Container title={"Execution Details"}/>
          <Container title={"Standard Output"}/>
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
