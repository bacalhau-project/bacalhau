// src/pages/Settings.tsx
import React, { useState } from "react";
import {
  useTableSettings,
  TableSettings,
} from "../context/TableSettingsContext";
import Layout from "../components/Layout";
import Checkbox from "../components/Checkbox";

const Settings = () => {
  const { settings, toggleSetting } = useTableSettings();
  const [tempSettings, setTempSettings] = useState(settings);

  const handleToggle = (settingKey: keyof TableSettings) => {
    setTempSettings((prev) => ({
      ...prev,
      [settingKey]: !prev[settingKey],
    }));
  };

  const handleSave = () => {
    (Object.keys(tempSettings) as Array<keyof TableSettings>).forEach((key) => {
      console.log(`Setting ${key} saved as ${tempSettings[key]}`);
      if (tempSettings[key] !== settings[key]) {
        toggleSetting(key);
        localStorage.setItem("tableSettings", JSON.stringify(tempSettings));
      }
    });
  };

  return (
    <Layout pageTitle="Settings">
      <Checkbox
        label="Show Node ID"
        checked={tempSettings.showNodeId}
        onChange={() => handleToggle("showNodeId")}
      />
      <button onClick={handleSave}>Save</button>
    </Layout>
  );
};

export default Settings;
