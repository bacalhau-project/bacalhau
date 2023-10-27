// src/components/ProgramSummary.tsx

import React from "react";
import styles from "../../styles/ProgramSummary.module.scss";
import cogwheel from "../../images/cogwheel-dark.png";
import Image from "next/image";

interface ProgramSummaryProps {
  data: string;
}

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  return (
    <div className={styles.programSummary}>
      <div className={styles.logo}>
        {/* Logo goes here */}
        <Image src={cogwheel} alt="Logo" width={20} height={20} />
      </div>
      <div className={styles.text}>
        <div className={styles.header}>{data[0]}</div>
        <div>{data[1]}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
