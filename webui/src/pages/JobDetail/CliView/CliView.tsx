import React from 'react';
import styles from "./CliView.module.scss";

interface CliViewProps {
    data?: string;
}

const CliView: React.FC<CliViewProps> = ({ data = "" }) => {
    return (
        <div className={styles.cliView}>
            {data !== "" ? data : "λ"}
        </div>
    );
};

export default CliView;
