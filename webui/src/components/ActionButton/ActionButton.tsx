import React from "react";
import styles from "./ActionButton.module.scss";
import { ReactComponent as ViewIcon } from "../../images/view-icon.svg";

interface ActionButtonProps {
  text: string;
}

const ActionButton: React.FC<ActionButtonProps> = ({ text }) => {
  return (
    <div className={styles.column}>
      <button className={styles.actionButton}>
        <ViewIcon/>
        {text}
      </button>
    </div>
  );
};

export default ActionButton;
