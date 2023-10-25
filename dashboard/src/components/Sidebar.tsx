// components/Sidebar.tsx
import React from 'react';
import styles from '../../styles/Sidebar.module.scss';

interface SidebarProps {
    isCollapsed: boolean;
    toggleSidebar: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ isCollapsed, toggleSidebar }) => {
    return (
        <div className={`${styles.sidebar} ${isCollapsed ? styles.collapsed : styles.expanded}`}>
            <button onClick={toggleSidebar}>
                {/* Toggle Icon */}
                {isCollapsed ? '>' : '<'}
            </button>
            <div className={styles.menu}>
                <div className={styles.menuItem}>
                    {/* Jobs Dashboard Icon */}
                    {!isCollapsed && 'Jobs Dashboard'}
                </div>
                <div className={styles.menuItem}>
                    {/* Nodes Dashboard Icon */}
                    {!isCollapsed && 'Nodes Dashboard'}
                </div>
            </div>
            <div className={styles.settings}>
                {/* Settings Icon */}
                {!isCollapsed && 'Settings'}
            </div>
        </div>
    );
};

export default Sidebar;