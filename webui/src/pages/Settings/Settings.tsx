import React, { useState } from "react";
import {
  useTableSettings,
  TableSettings,
} from "../../context/TableSettingsContext";
import Layout from "../../layout/Layout";
import Checkbox from "../../components/Checkbox/Checkbox";

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
      if (tempSettings[key] !== settings[key]) {
        toggleSetting(key);
        localStorage.setItem("tableSettings", JSON.stringify(tempSettings));
      }
    });
  };

  return (
    <Layout pageTitle="Settings">
      {/* Node Dashboard Settings */}
      <h3>Nodes Dashboard Settings:</h3>
      <Checkbox
        label="Node ID"
        checked={tempSettings.showNodeId}
        onChange={() => handleToggle("showNodeId")}
      />
      <Checkbox
        label="Name"
        checked={tempSettings.showName}
        onChange={() => handleToggle("showName")}
      />
      <Checkbox
        label="Type"
        checked={tempSettings.showType}
        onChange={() => handleToggle("showType")}
      />
      <Checkbox
        label="Environment"
        checked={tempSettings.showEnv}
        onChange={() => handleToggle("showEnv")}
      />
      <Checkbox
        label="Inputs In"
        checked={tempSettings.showInputs}
        onChange={() => handleToggle("showInputs")}
      />
      <Checkbox
        label="Outputs"
        checked={tempSettings.showOutputs}
        onChange={() => handleToggle("showOutputs")}
      />
      <Checkbox
        label="Version"
        checked={tempSettings.showVersion}
        onChange={() => handleToggle("showVersion")}
      />
      <Checkbox
        label="Action"
        checked={tempSettings.showAction}
        onChange={() => handleToggle("showAction")}
      />
      <button onClick={handleSave}>Save</button>
    </Layout>
  );
};

export default Settings;
