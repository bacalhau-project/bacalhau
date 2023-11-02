// src/pages/JobsDashboard.tsx

import React, { useEffect, useState } from "react";
import styles from "../../styles/JobsDashboard.module.scss";
import JobsTable from "../components/JobsTable";
import Layout from "../components/Layout";
import { Job } from "../interfaces";
import { list } from "../../../../nodejs-sdk/src/sdk/api"; //TODO: Temporary import of NodeJS SDK

const JobsDashboard: React.FC = () => {
  const [data, setData] = useState<{ jobs: Job[] }>({ jobs: [] });

  async function getJobsData() {
    try {
      const listData = await list();
      setData(listData);
    } catch (error) {
      console.error(error);
    }
  }

  useEffect(() => {
    getJobsData();
  }, []);

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsdashboard}>
        <JobsTable data={data.jobs} />
      </div>
    </Layout>
  );
};

export default JobsDashboard;
