import React from 'react';
import styles from './Container.module.scss';

interface ContainerProps {
  title: string;
  children?: React.ReactNode;
}

const Container: React.FC<ContainerProps> = ({ title, children }) => (
  <div className={styles.container}>
    <div className={styles.title}>
      <span className={styles.titleLine} />
      {title}
    </div>
    <div className={styles.content}>{children}</div>
  </div>
);

export default Container;
