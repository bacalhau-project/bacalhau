import React from "react";
import Image from "next/image";
import styles from "../../styles/ActionButton.module.scss";
import icon from "../public/images/view-icon.png";

interface ActionButtonProps {
  text: string;
}

const ActionButton: React.FC<ActionButtonProps> = ({ text }) => {
  return (
    <div className={styles.column}>
      <button className={styles.actionButton}>
        <Image src={icon} alt="Logo" width={20} />
        {text}
      </button>
    </div>
  );
};

export default ActionButton;
