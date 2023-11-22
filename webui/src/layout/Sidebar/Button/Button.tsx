// Sidebar/Button.tsx
import React from "react";
import styles from "./Button.module.scss";

interface ButtonProps {
  toggleSidebar: () => void;
  isCollapsed: boolean;
}

const Button: React.FC<ButtonProps> = ({
  toggleSidebar,
  isCollapsed,
}) => {
  return (
    <section className={styles.mb3}>
      <nav className={`${styles.navbar} ${styles.navbarInfo} ${styles.bgInfo}`}>
        <div className={styles.containerFluid}>
          <button
            className={`${styles.navbarToggler} ${styles.thirdButton}`}
            type="button"
            onClick={toggleSidebar}
            aria-controls="navbarToggleExternalContent11"
            aria-expanded="false"
            aria-label="Toggle navigation"
          >
            <div
              className={`${styles.animatedIcon3} ${
                !isCollapsed ? styles.open : ""
              }`}
            >
              <span></span>
              <span></span>
              <span></span>
            </div>
          </button>
        </div>
      </nav>
    </section>
  );
};

export default Button;
