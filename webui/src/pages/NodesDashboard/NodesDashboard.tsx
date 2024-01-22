import React, { useEffect, useState } from "react";
import styles from "./NodesDashboard.module.scss";
import NodesTable from "./NodesTable/NodesTable";
import Layout from "../../layout/Layout";
import { Node } from "../../helpers/nodeInterfaces";
import { bacalhauAPI } from "../../services/bacalhau";
import { useTableSettings } from "../../context/TableSettingsContext";

const NodesDashboard: React.FC = () => {
  const [data, setData] = useState<Node[]>([]);
  const { settings } = useTableSettings();

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
        <NodesTable key={JSON.stringify(settings)} data={data} />
      </div>
    </Layout>
  );
};

export default NodesDashboard;
