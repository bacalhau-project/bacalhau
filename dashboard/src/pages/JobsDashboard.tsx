// src/pages/JobsDashboard.tsx

import React, { useEffect, useState } from "react";
import styles from "../../styles/JobsDashboard.module.scss";
import Table from "../components/Table";
import Layout from "../components/Layout";
import { fetchList } from './utils/sdkWrapper';
// import { getJobs, Resolved } from "./api/bacalhau";
const {
	getClientId,
	submit,
	list,
	results,
	states,
	events,
} = require("../../../../nodejs-sdk/src/sdk/api");

const JobsDashboard: React.FC = () => {
  const headers = [
    "Job ID",
    "Name",
    "Created",
    "Program",
    "Job Type",
    "Label",
    "Status",
    "Action",
  ];
  const data = [
    [
      "xxxxxxxx",
      "Long Running Job #1",
      "2 minutes ago",
      ["Ubuntu", "Hello World!"],
      "Daemon",
      "Canary",
      "Running",
      "View",
    ],
    [
      "xxxxxxxx",
      "",
      "October 20, 2023",
      ["Ubuntu", "Hello World!"],
      "Batch",
      "",
      "Complete",
      "View",
    ],
  ];

  // const [data2, setData] = useState<MyType[]>([]);

  // useEffect(() => {
  //   async function getList() {
  //     try {
  //       const listData = await fetchList();
  //       setData(listData);
  //     } catch (error) {
  //       console.error(error);
  //     }
  //   }

  //   getList();
  // }, []);

  const [jobs, setJobs] = useState<any[]>([]);  // Adjust the type of jobs based on your data structure
  const [loading, setLoading] = useState<boolean>(true);

  async function getJobsData() {
    const listttt = await list()
    console.log("HIIII", listttt);  // Log the data here
  }

  useEffect(() => {
    getJobsData();
  }, []);

  return (
    <Layout pageTitle="Jobs Dashboard">
      <div className={styles.jobsdashboard}>
        <Table headers={headers} data={data} />
        {/* {data2 && data2.map(item => (
          <div key={item.id}>{item.name}</div>
        ))} */}
      </div>
    </Layout>
  );
};

export default JobsDashboard;
