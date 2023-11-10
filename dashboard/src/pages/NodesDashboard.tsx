// src/pages/JobsDashboard.tsx

import React from "react";
import styles from "../../styles/NodesDashboard.module.scss";
import Layout from "../components/Layout";

const NodesDashboard: React.FC = () => {
  return (
    <Layout pageTitle="Nodes Dashboard">
      <div className={styles.nodesdashboard}></div>
    </Layout>
  );
};

export default NodesDashboard;
