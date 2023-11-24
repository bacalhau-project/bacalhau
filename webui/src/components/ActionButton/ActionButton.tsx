import React from "react";
import styles from "./ActionButton.module.scss";
import icon from "../../images/view-icon.png";

interface ActionButtonProps {
  text: string;
}

const ActionButton: React.FC<ActionButtonProps> = ({ text }) => {
  return (
    <div className={styles.column}>
      <button className={styles.actionButton}>
        <img src={icon} alt="icon" width={20}/>
        {text}
      </button>
    </div>
  );
};

export default ActionButton;
