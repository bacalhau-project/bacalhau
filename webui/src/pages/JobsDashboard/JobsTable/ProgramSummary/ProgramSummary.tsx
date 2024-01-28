// src/components/ProgramSummary.tsx
import React from "react";
import styles from "./ProgramSummary.module.scss";
import { Task } from "../../../../helpers/jobInterfaces";
import { SVGImage } from "../../../../images/svg-image"


interface ProgramSummaryProps {
  data: Task;
}

const getImageSource = (type: string) => {
  switch (type) {
    case "docker":
      return (
        <SVGImage
          src="../../images/docker.svg"
          alt="Docker"
          svgClassName={styles.icon}
        />
      )
    default:
      return (
        <SVGImage
          src="../../images/cogwheel.svg"
          alt="Settings"
          svgClassName={styles.icon}
        />
      )
  }
};

const truncateInput = (text: string, length: number) => {
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
  const truncatedInput = truncateInput((parameters || []).join(" "), 100);

  return (
    <div className={styles.programSummary}>
      <div className={styles.iconContainer}>{imageSource}</div>
      <div className={styles.text}>
        <div className={styles.header}>{image}</div>
        <div className={styles.input}>{truncatedInput}</div>
      </div>
    </div>
  );
};

export default ProgramSummary;
