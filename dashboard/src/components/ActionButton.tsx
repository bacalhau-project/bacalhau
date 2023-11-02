import React from 'react';
import Image from "next/image";
import styles from "../../styles/ActionButton.module.scss";
import icon from "../../images/view-icon.png"; 

interface ActionButtonProps {
    text: string;
}

const ActionButton: React.FC<ActionButtonProps> = ({ text }) => {
    return (
        <button className={styles.actionButton}>
            <Image src={icon} alt="Logo" width={20} />
            {text}
        </button>
    );
}

export default ActionButton;
