// components/Sidebar.tsx

import React from "react";
import styles from "../../styles/Sidebar.module.scss";
import SidebarButton from "./SidebarButton";
import Link from "next/link";
import Image from "next/image";
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
      <SidebarButton toggleSidebar={toggleSidebar} isCollapsed={isCollapsed} />
      <div className={styles.menu}>
        <div className={styles.menuItem}>
          {/* Jobs Dashboard Icon */}
          <Link href="/JobsDashboard" className={styles.link}>
            <Image src={jobsIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Jobs Dashboard</span>
          </Link>
        </div>
        <div className={styles.menuItem}>
          {/* Nodes Dashboard Icon */}
          <Link href="/NodesDashboard" className={styles.link}>
            <Image src={nodesIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Nodes Dashboard</span>
          </Link>
        </div>
        <div className={styles.menuItem}>
          {/* Settings Icon */}
          <Image src={settingsIcon} alt="Icon" width={20} height={20} />
          <span className={styles.menuText}>Settings</span>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;
