// layout/Sidebar/Sidebar.tsx
import React from "react"
import { Link } from "react-router-dom"
import { SVGImage } from "../../images/svg-image"
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
      icon: <SVGImage src="../../images/jobs-icon.svg" alt="Jobs" />,
      title: "Jobs Dashboard",
    },
    {
      path: "/NodesDashboard",
      icon: <SVGImage src="../../images/nodes-icon.svg" alt="Nodes" />,
      title: "Nodes Dashboard",
    },
    {
      path: "/Settings",
      icon: <SVGImage src="../../images/cogwheel.svg" alt="Settings" />,
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
        <SVGImage src="../../images/bacalhau.svg" alt="Bacalhau Icon" />
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
