// components/Sidebar.tsx
import React from "react";
import { Link } from 'react-router-dom'
import styles from "./Sidebar.module.scss";
import Button from "./Button/Button";
import jobsIcon from "../../images/jobs-icon.png";
import nodesIcon from "../../images/nodes-icon.png";
import settingsIcon from "../../images/settings-icon.png";

interface SidebarProps {
  isCollapsed: boolean;
  toggleSidebar: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ isCollapsed, toggleSidebar }) => {
  return (
    <div
      className={`${styles.sidebar} ${
        isCollapsed ? styles.collapsed : styles.expanded
      }`}
    >
      <Button toggleSidebar={toggleSidebar} isCollapsed={isCollapsed} />
      <div className={styles.menu}>
        <div className={styles.menuItem}>
          {/* Jobs Dashboard Icon */}
          <Link to="/JobsDashboard" className={styles.link}>
            <img src={jobsIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Jobs Dashboard</span>
          </Link>
        </div>
        <div className={styles.menuItem}>
          {/* Nodes Dashboard Icon */}
          <Link to="/NodesDashboard" className={styles.link}>
            <img src={nodesIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Nodes Dashboard</span>
          </Link>
        </div>
        <div className={styles.menuItem}>
          {/* Settings Icon */}
          <Link to="/Settings" className={styles.link}>
            <img src={settingsIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Settings</span>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;
