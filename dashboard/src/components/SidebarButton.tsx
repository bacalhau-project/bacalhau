// components/SidebarButton.tsx
import React from 'react';
import styles from '../../styles/SidebarButton.module.scss';

interface SidebarButtonProps {
  toggleSidebar: () => void;
  isCollapsed: boolean; 
}

const SidebarButton: React.FC<SidebarButtonProps> = ({ toggleSidebar, isCollapsed }) => {
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
            <div className={`${styles.animatedIcon3} ${!isCollapsed ? styles.open : ''}`}>
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

export default SidebarButton;
