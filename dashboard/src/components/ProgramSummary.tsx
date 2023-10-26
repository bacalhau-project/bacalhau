// src/components/ProgramSummary.tsx

import React from "react";
import styles from "../../styles/ProgramSummary.module.scss";

interface ProgramSummaryProps {
  data: string; // Update this type based on the data you're passing
}

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  return (
    <div className={styles.programSummary}>
      <div className={styles.logo}>
        {/* Logo goes here */}
        <img src="/logo.png" alt="Logo" />
      </div>
      <div className={styles.text}>
        <div className={styles.header}>Header</div>
        <div>{data}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
