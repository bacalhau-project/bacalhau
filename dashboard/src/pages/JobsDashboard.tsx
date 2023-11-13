// src/pages/JobsDashboard.tsx

import React, { useEffect, useState } from "react";
import styles from "../../styles/JobsDashboard.module.scss";
import JobsTable from "../components/JobsTable";
import Layout from "../components/Layout";
import { Job } from "../helpers/interfaces";
import { bacalhauAPI } from "./api/bacalhau";

const JobsDashboard: React.FC = () => {
  const [data, setData] = useState<Job[]>([]);

  async function getJobsData() {
    try {
      const response = await bacalhauAPI.listJobs();
      if (response.Jobs) {
        setData(response.Jobs);
      }
    } catch (error) {
      console.error(error);
    }
  }

  useEffect(() => {
    getJobsData();
  }, []);

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsDashboard}>
        <JobsTable data={data} />
      </div>
    </Layout>
  );
};

export default JobsDashboard;
