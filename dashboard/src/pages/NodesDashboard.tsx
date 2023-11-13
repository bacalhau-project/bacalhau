// src/pages/NodesDashboard.tsx

import React, { useEffect, useState } from "react";
import styles from "../../styles/NodesDashboard.module.scss";
import JobsTable from "../components/NodesTable";
import Layout from "../components/Layout";
import { Node } from "../helpers/nodeInterfaces";
import { bacalhauAPI } from "./api/bacalhau";

const NodesDashboard: React.FC = () => {
  const [data, setData] = useState<Node[]>([]);

  async function getNodesData() {
    try {
      const response = await bacalhauAPI.listNodes();
      if (response.Nodes) {
        setData(response.Nodes);
      }
    } catch (error) {
      console.error(error);
    }
  }

  useEffect(() => {
    getNodesData();
  }, []);

  return (
    <Layout pageTitle="Nodes Dashboard">
      <div className={styles.nodesDashboard}>
        <JobsTable data={data} />
      </div>
    </Layout>
  );
};

export default NodesDashboard;
