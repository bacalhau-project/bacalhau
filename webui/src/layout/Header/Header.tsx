import React from "react"
import { SVGImage } from "../../images/svg-image"
import styles from "./Header.module.scss"

interface HeaderProps {
  collapsed: boolean
  pageTitle: string
}

export const Header: React.FC<HeaderProps> = ({ pageTitle, collapsed }) => (
  <header className={styles.header} data-collapsed={collapsed}>
    <div className={styles.left}>
      <SVGImage
        src="/images/bacalhau.svg"
        alt="Bacalhau Icon"
        svgClassName={styles.headerLogo}
      />
      <div className={styles.pageTitle}>
        <h1 aria-label={pageTitle} className={styles.pageTitle}>
          {pageTitle}
        </h1>
      </div>
      <div className={styles.searchBar}>
        {/* Placeholder for search bar */}
        {/* <input type="text" placeholder="Search..." /> */}
      </div>
    </div>
    <div className={styles.right}>
      {/* Profile section */}
      <div className={styles.profile}>
        <SVGImage src="/images/profile.svg" alt="Profile Icon" />
      </div>
    </div>
  </header>
)
