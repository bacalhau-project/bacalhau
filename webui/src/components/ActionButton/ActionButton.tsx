import React from "react";
import { useNavigate } from 'react-router-dom';
import styles from "./ActionButton.module.scss";
import icon from "../../images/view-icon.png";

interface ActionButtonProps {
  text: string;
  onClick?: () => void; // Optional, if you want to handle the click within the parent component
  to?: string; // Optional, path to navigate to
  id?: string; // Optional, id
}

const ActionButton: React.FC<ActionButtonProps> = ({ text, onClick, to, id }) => {
  const navigate = useNavigate();

  const handleClick = () => {
    // If a path is provided, navigate to that path, appending jobId if it exists
    if (to) {
      const path = id ? `${to}/${id}` : to;
      navigate(path);
    }
  };

  return (
    <div className={styles.column}>
      <button className={styles.actionButton} onClick={handleClick}>
        <img src={icon} alt="icon" width={20}/>
        {text}
      </button>
    </div>
  );
};

export default ActionButton;
