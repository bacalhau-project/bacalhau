// layout/Sidebar/Sidebar.tsx
import React from "react"
import { Link } from "react-router-dom"
import styles from "./Sidebar.module.scss"
import Button from "./Button/Button"
import { ReactComponent as JobsIcon } from "../../images/jobs-icon.svg"
import { ReactComponent as NodesIcon } from "../../images/nodes-icon.svg"
import { ReactComponent as CogWheelIcon } from "../../images/cogwheel.svg"
import { ReactComponent as BacalhauIcon } from "../../images/bacalhau.svg"

export const toggleSidebarFn = () => {
  const sidebar = document.querySelector(`.${styles.sidebar}`)
  if (sidebar) {
    // If sidebar is expanded, collapse it
    if (!sidebar.classList.contains(styles.collapsed)) {
      sidebar.classList.remove(styles.collapsed)
      sidebar.classList.add(styles.expanded)
    } else {
      // If sidebar is collapsed, expand it
      sidebar.classList.add(styles.collapsed)
      sidebar.classList.remove(styles.expanded)
    }
  }
}

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
      icon: <CogWheelIcon />,
      title: "Settings",
    },
  ]

  return (
    <div
      className={`${styles.sidebar} 
      ${isCollapsed ? styles.collapsed : styles.expanded}`}
    >
      <div className={styles.header}>
        <Button toggleSidebar={toggleSidebar} isCollapsed={isCollapsed} />
        <BacalhauIcon height="24" width="" />
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
