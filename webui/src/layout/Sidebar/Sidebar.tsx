// layout/Sidebar/Sidebar.tsx
import React from "react"
import { Link } from "react-router-dom"
import { ReactComponent as BacalhauIcon } from "../../images/bacalhau.svg"
import { ReactComponent as JobsIcon } from "../../images/jobs-icon.svg"
import { ReactComponent as NodesIcon } from "../../images/nodes-icon.svg"
import { ReactComponent as CogwheelIcon } from "../../images/cogwheel.svg"
import styles from "./Sidebar.module.scss"
import { Button } from "./Button/Button"

interface SidebarProps {
  isCollapsed: boolean
  toggleSidebar: () => void
}

export const Sidebar: React.FC<SidebarProps> = ({
  isCollapsed,
  toggleSidebar,
}) => {
  const links = [
    {
      path: "/JobsDashboard",
      icon: <JobsIcon />,
      title: "Jobs Dashboard",
    },
    {
      path: "/NodesDashboard",
      icon: <NodesIcon />,
      title: "Nodes Dashboard",
    },
    {
      path: "/Settings",
      icon: <CogwheelIcon />,
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
        <BacalhauIcon />
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
  )
}
