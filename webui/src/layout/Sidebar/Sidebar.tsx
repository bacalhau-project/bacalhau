// layout/Sidebar/Sidebar.tsx
import React from "react";
import { Link } from "react-router-dom";
import styles from "./Sidebar.module.scss";
import Button from "./Button/Button";
import { ReactSVG } from "react-svg";

interface SidebarProps {
  isCollapsed: boolean;
  toggleSidebar: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  isCollapsed,
  toggleSidebar,
}) => {
  const links = [
    {
      path: "/JobsDashboard",
      icon: <ReactSVG src="../../images/jobs-icon.svg" />,
      title: "Jobs Dashboard",
    },
    {
      path: "/NodesDashboard",
      icon: <ReactSVG src="../../images/nodes-icon.svg" />,
      title: "Nodes Dashboard",
    },
    {
      path: "/Settings",
      icon: <ReactSVG src="../../images/cogwheel.svg" />,
      title: "Settings",
    },
  ];

  return (
    <div
      className={`${styles.sidebar} ${
        isCollapsed ? styles.collapsed : styles.expanded
      }`}
    >
      <div className={styles.header}>
        <Button toggleSidebar={toggleSidebar} isCollapsed={isCollapsed} />
        <ReactSVG src="../../images/bacalhau.svg" height="24" width="" />
      </div>
      <div className={styles.menu}>
        {links.map((link) => (
          <Link
            key={link.title}
            to={link.path}
            className={styles.menuItem}
            data-selected={document.location.pathname.startsWith(link.path)}
            title={link.title}
          >
            {link.icon}
            <span className={styles.menuText}>{link.title}</span>
          </Link>
        ))}
      </div>
    </div>
  );
};
