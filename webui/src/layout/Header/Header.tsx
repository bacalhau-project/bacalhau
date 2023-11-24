import React from "react";
import styles from "./Header.module.scss";
import bacLogo from "../../images/bacalhau-logo-black.png";
import profile from "../../images/profile.png";

interface HeaderProps {
  pageTitle: string;
}

const Header: React.FC<HeaderProps> = ({ pageTitle }) => {
  return (
    <header className={styles.header}>
      <div className={styles.left}>
        <div className={styles.logo}>
          <img src={bacLogo} alt="Logo" width={150} height={150} />
        </div>
        <div className={styles.pageTitle}>{pageTitle}</div>
        <div className={styles.searchBar}>
          {/* Placeholder for search bar */}
          {/* <input type="text" placeholder="Search..." /> */}
        </div>
      </div>
      <div className={styles.right}>
        {/* Profile section */}
        <div className={styles.profile}>
          <img src={profile} alt="Logo" width={150} height={150} />
        </div>
      </div>
    </header>
  );
};

export default Header;
