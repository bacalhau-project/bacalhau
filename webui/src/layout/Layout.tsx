// components/Layout.tsx
import React, { useState } from "react";
import styles from "./Layout.module.scss";
import Header from "./Header/Header";
import Sidebar from "./Sidebar/Sidebar";

interface LayoutProps {
  pageTitle: string;
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ pageTitle, children }) => {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(true);
  const toggleSidebar = () => setIsSidebarCollapsed(!isSidebarCollapsed);

  return (
    <div className={styles.layout}>
      <Sidebar isCollapsed={isSidebarCollapsed} toggleSidebar={toggleSidebar} />
      <div
        className={`${styles.rightSide} ${
          isSidebarCollapsed ? "" : styles.expandedSidebar
        }`}
      >
        <Header pageTitle={pageTitle} />
        <main>{children}</main>
      </div>
    </div>
  );
};

export default Layout;
