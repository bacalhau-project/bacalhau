import { useState } from "react";
import {
  useTableSettings,
  TableSettings,
} from "../../context/TableSettingsContext";
import styles from "./Settings.module.scss";
import Layout from "../../layout/Layout";
import Checkbox from "../../components/Checkbox/Checkbox";
import Container from "../../components/Container/Container";

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

  const jobsOptions: { label: string; key: keyof TableSettings }[] = [
    { label: "Job ID", key: "showJobId" },
    { label: "Name", key: "showJobName" },
    { label: "Created", key: "showCreated" },
    { label: "Program", key: "showProgram" },
    { label: "Job Type", key: "showJobType" },
    { label: "Label", key: "showLabel" },
    { label: "Status", key: "showStatus" },
  ];

  const nodesOptions: { label: string; key: keyof TableSettings }[] = [
    { label: "Node ID", key: "showNodeId" },
    { label: "Name", key: "showNodeName" },
    { label: "Type", key: "showNodeType" },
    { label: "Environment", key: "showEnv" },
    { label: "Inputs In", key: "showInputs" },
    { label: "Outputs", key: "showOutputs" },
    { label: "Version", key: "showVersion" },
  ];

  return (
    <Layout pageTitle="Settings">
      {/* Jobs Dashboard Settings */}
      <Container title={"Jobs Dashboard Settings"}>
        {jobsOptions.map(({ label, key }) => (
          <div className={styles.checkboxGroup}>
            <Checkbox
              label={label}
              checked={tempSettings[key]}
              onChange={() => handleToggle(key)}
            />
          </div>
        ))}
        <button onClick={handleSave} className={styles.saveButton}>
          Save
        </button>
      </Container>
      {/* Node Dashboard Settings */}
      <Container title={"Nodes Dashboard Settings"}>
        {nodesOptions.map(({ label, key }) => (
          <div className={styles.checkboxGroup}>
            <Checkbox
              label={label}
              checked={tempSettings[key]}
              onChange={() => handleToggle(key)}
            />
          </div>
        ))}
        <button onClick={handleSave} className={styles.saveButton}>
          Save
        </button>
      </Container>
    </Layout>
  );
};

export default Settings;
