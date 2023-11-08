// src/pages/JobsDashboard.tsx

import React, { useEffect, useState } from "react";
import styles from "../../styles/JobsDashboard.module.scss";
import JobsTable from "../components/JobsTable";
import Layout from "../components/Layout";
import { Job } from "../interfaces";
// import { list } from "../../../../nodejs-sdk/src/sdk/api"; //TODO: Temporary import of NodeJS SDK
import { bacalhauAPI } from "./api/bacalhau";

const JobsDashboard: React.FC = () => {
  const [data, setData] = useState<Job[]>([]);
  const [error, setError] = useState<string | null>(null);

  async function getJobsData() {
    try {
      const listData = await bacalhauAPI.listJobs();
      setData(listData);
      console.log("JOBS", listData)
    } catch (error) {
      setError('Failed to fetch jobs');
      console.error(error);
    }
  }

  useEffect(() => {
    getJobsData();
  }, []);

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsdashboard}>
        {/* <JobsTable data={data.jobs} /> */}
      </div>
    </Layout>
  );
};

export default JobsDashboard;
