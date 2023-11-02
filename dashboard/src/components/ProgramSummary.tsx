// src/components/ProgramSummary.tsx
import React from "react";
import styles from "../../styles/ProgramSummary.module.scss";
import cogwheel from "../../images/cogwheel-dark.png";
import Image from "next/image";
import { EngineSpec } from "../interfaces";

interface ProgramSummaryProps {
  data: EngineSpec;
}

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  const { Image: image, Parameters: input } = data.Params;
  console.log("data", data);
  console.log("image", image);
  console.log("input", input);

  return (
    <div className={styles.programSummary}>
      <div className={styles.logo}>
        <Image src={cogwheel} alt="Logo" width={20} height={20} />
      </div>
      <div className={styles.text}>
        <div className={styles.header}>{image}</div>
        <div className={styles.input}>{input}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
