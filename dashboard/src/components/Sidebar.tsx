// components/Sidebar.tsx

import React from "react";
import styles from "../../styles/Sidebar.module.scss";
import SidebarButton from './SidebarButton';
import Link from 'next/link';

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
                    <Link href="/JobsDashboard" className={styles.link}>Jobs Dashboard</Link>  
                </div>
                <div className={styles.menuItem}>
                    {/* Nodes Dashboard Icon */}
                    <Link href="/NodesDashboard" className={styles.link}>Nodes Dashboard</Link>  
                </div>
            </div>
            <div className={styles.settings}>
                {/* Settings Icon */}
                Settings
            </div>
        </div>
    );
};

export default Sidebar;
