// src/components/ProgramSummary.tsx
import React from "react";
import styles from "../../styles/ProgramSummary.module.scss";
import Image from "next/image";
import { EngineSpec } from "../interfaces";

// Image imports
import cogwheel from "../../images/cogwheel-dark.png";
import dockerImage from "../../images/docker.png";

interface ProgramSummaryProps {
  data: EngineSpec;
}

const getImageSource = (type: string) => {
  switch (type) {
    case "docker":
      return dockerImage;
    default:
      return cogwheel;
  }
};

const truncateInput = (text: string[], length: number) => {
  if (text[0].length <= length) return text;
  return text[0].substring(0, length) + "[cont]";
};

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  const { Image: image, Parameters: input } = data.Params;
  const imageSource = getImageSource(data.Type);
  return (
    <div className={styles.programSummary}>
      <div className={styles.logo}>
        <Image src={imageSource} alt="Logo" width={20} height={20} />
      </div>
      <div className={styles.text}>
        <div className={styles.header}>{image}</div>
        <div className={styles.input}>{truncateInput(input, 100)}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
