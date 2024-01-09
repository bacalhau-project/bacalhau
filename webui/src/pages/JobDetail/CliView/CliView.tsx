import React from 'react';
import styles from './CliView.module.scss';

interface CliViewProps {
  data?: string;
}

// Limit the number of lines
const limitLines = (inputData: string, maxLines: number) => {
  const lines = inputData.split('\n');
  return lines.slice(0, maxLines).join('\n');
};

const CliView: React.FC<CliViewProps> = ({ data = '' }) => {
  // Limiting the data to 2000 lines
  const displayData = data !== '' ? limitLines(data, 2000) : 'λ';

  return <div className={styles.cliView}>{displayData}</div>;
};

export default CliView;
