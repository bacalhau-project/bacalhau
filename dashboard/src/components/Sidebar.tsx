// components/Sidebar.tsx
import React from "react";
import Link from "next/link";
import Image from "next/image";
import styles from "../../styles/Sidebar.module.scss";
import SidebarButton from "./SidebarButton";
import jobsIcon from "../public/images/jobs-icon.png";
import nodesIcon from "../public/images/nodes-icon.png";
import settingsIcon from "../public/images/settings-icon.png";

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
          <Link href="/Settings" className={styles.link}>
            <Image src={settingsIcon} alt="Icon" width={20} height={20} />
            <span className={styles.menuText}>Settings</span>
          </Link>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;
