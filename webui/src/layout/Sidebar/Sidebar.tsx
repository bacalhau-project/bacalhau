// components/Sidebar.tsx
import React from "react";
import { Link } from 'react-router-dom'
import styles from "./Sidebar.module.scss";
import Button from "./Button/Button";
import { ReactComponent as BacalhauLogo} from "../../images/bacalhau.svg";
import jobsIcon from "../../images/jobs-icon.png";
import nodesIcon from "../../images/nodes-icon.png";
import settingsIcon from "../../images/settings-icon.png";

interface SidebarProps {
  isCollapsed: boolean;
  toggleSidebar: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ isCollapsed, toggleSidebar }) => {
  const links = [
    {
      path: "/JobsDashboard",
      icon: jobsIcon,
      title: "Jobs Dashboard",
    }, {
      path: "/NodesDashboard",
      icon: nodesIcon,
      title: "Nodes Dashboard",
    }, {
      path: "/Settings",
      icon: settingsIcon,
      title: "Settings",
    },
  ]

  return (
    <div
      className={`${styles.sidebar} ${
        isCollapsed ? styles.collapsed : styles.expanded
      }`}
    >
      <div className={styles.header}>
        <Button toggleSidebar={toggleSidebar} isCollapsed={isCollapsed} />
        <BacalhauLogo height="24" width="" />
      </div>
      <div className={styles.menu}>
        {links.map(link =>
          <Link key={link.title} to={link.path} className={styles.menuItem} data-selected={document.location.pathname.startsWith(link.path)} title={link.title}>
            <img src={link.icon} alt="" width={20} height={20} />
            <span className={styles.menuText}>{link.title}</span>
          </Link>
        )}
      </div>
    </div>
  );
};

export default Sidebar;
