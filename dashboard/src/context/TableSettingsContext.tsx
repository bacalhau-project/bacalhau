// src/context/TableSettingsContext.tsx
import React, { useState, useContext, useEffect, ReactNode } from "react";

export interface TableSettings {
  showNodeId: boolean;
  // ... add other settings as needed
}

interface TableSettingsContextType {
  settings: TableSettings;
  toggleSetting: (key: keyof TableSettings) => void;
}

const defaultState: TableSettings = {
  showNodeId: true,
  // ... default values for other settings
};

const defaultContextValue: TableSettingsContextType = {
  settings: defaultState,
  toggleSetting: () => {},
};

const TableSettingsContext =
  React.createContext<TableSettingsContextType>(defaultContextValue);

export const TableSettingsProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [settings, setSettings] = useState<TableSettings>(defaultState);

  useEffect(() => {
    const loadSettings = () => {
      const storedSettings = localStorage.getItem("tableSettings");
      if (storedSettings) {
        setSettings(JSON.parse(storedSettings));
      }
    };

    if (typeof window !== "undefined") {
      loadSettings();
    }
  }, []);

  const toggleSetting = (key: keyof TableSettings) => {
    setSettings((prev) => {
      const newSettings = { ...prev, [key]: !prev[key] };
      localStorage.setItem("tableSettings", JSON.stringify(newSettings));
      console.log("newSettings", newSettings);
      return newSettings;
    });
  };

  return (
    <TableSettingsContext.Provider value={{ settings, toggleSetting }}>
      {children}
    </TableSettingsContext.Provider>
  );
};

export const useTableSettings = (): TableSettingsContextType => {
  console.log("IM HERE");

  const context = useContext(TableSettingsContext);
  if (!context) {
    throw new Error(
      "useTableSettings must be used within a TableSettingsProvider",
    );
  }
  return context;
};

export default TableSettingsContext;
