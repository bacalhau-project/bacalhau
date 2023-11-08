import React from "react";
import styles from "../../styles/Label.module.scss";

interface LabelProps {
  text: string;
  color: string;
}

const Label: React.FC<LabelProps> = ({ text, color }) => {
  const labelClass = `${styles.label} ${styles[`label-${color}`] || ""}`;
  return (
    <div className={styles.column}>
      <button className={labelClass}>{text}</button>
    </div>
  );
};

export default Label;
