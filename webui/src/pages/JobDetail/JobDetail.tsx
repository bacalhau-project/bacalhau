import React, { useEffect, useState } from "react";
import styles from "./JobDetail.module.scss";
import Layout from "../../layout/Layout";

const JobDetail: React.FC = () => {
  return (
    <Layout pageTitle="Job Detail">
      <div className={styles.jobDetail}>
      </div>
    </Layout>
  );
};

export default JobDetail;
