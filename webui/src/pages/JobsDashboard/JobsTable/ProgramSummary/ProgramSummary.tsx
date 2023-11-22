// src/components/ProgramSummary.tsx
import React from "react";
import styles from "./ProgramSummary.module.scss";
import { Tasks } from "../../../../helpers/jobInterfaces";
import cogwheel from "../../../../images/cogwheel-dark.png";
import dockerImage from "../../../../images/docker.png";

interface ProgramSummaryProps {
  data: Tasks;
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
  if (text.length === 0) return "";
  if (text[0].length <= length) return text;
  return text[0].substring(0, length) + "[cont]";
};

const ProgramSummary: React.FC<ProgramSummaryProps> = ({ data }) => {
  const {
    Type: engineType,
    Params: { Image: image, Parameters: parameters },
  } = data.Engine;

  const imageSource = getImageSource(engineType);
  const truncatedInput = truncateInput(parameters || [], 100);

  return (
    <div className={styles.programSummary}>
      <div className={styles.logo}>
        <img src={imageSource} alt="Engine" width={20} height={20} />
      </div>
      <div className={styles.text}>
        <div className={styles.header}>{image}</div>
        <div className={styles.input}>{truncatedInput}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
