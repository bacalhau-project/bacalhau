// src/components/NodesTable.tsx
import React, { useContext } from "react";
import styles from "../../styles/NodesTable.module.scss";
import TableSettingsContext from "../context/TableSettingsContext";
import Label from "./Label";
import ActionButton from "./ActionButton";
import { Node, ParsedNodeData } from "../helpers/nodeInterfaces";

interface TableProps {
  data: Node[];
}

// const labelColorMap: { [key: string]: string } = {
//   healthy: "green",
//   warning: "orange",
//   critical: "red",
//   offline: "blue",
//   unknown: "grey"
// };

function parseData(nodes: Node[]): ParsedNodeData[] {
  return nodes.map((node) => {
    const inputs: string[] = node.ComputeNodeInfo?.StorageSources ?? [];
    const outputs: string[] = node.ComputeNodeInfo?.Publishers ?? [];

    return {
      id: node.PeerInfo.ID,
      name: node.Labels.name ? node.Labels.name : node.PeerInfo.ID,
      type: node.NodeType,
      environment: node.Labels.env,
      inputs: inputs,
      outputs: outputs,
      version: node.BacalhauVersion.GitVersion,
      action: "Action",
    };
  });
}

const NodesTable: React.FC<TableProps> = ({ data }) => {
  const parsedData = parseData(data);
  const { settings } = useContext(TableSettingsContext);
  console.log("settings.showNodeId", settings.showNodeId);
  return (
    <div className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            {settings.showNodeId && <th>Node ID</th>}
            <th>Name</th>
            <th>Type</th>
            <th>Environment</th>
            <th>Inputs From</th>
            <th>Outputs</th>
            <th>Version</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {parsedData.map((nodeData, index) => (
            <tr key={index}>
              {settings.showNodeId && (
                <td className={styles.id}>{nodeData.id}</td>
              )}
              <td className={styles.name}>{nodeData.name}</td>
              <td className={styles.type}>{nodeData.type}</td>
              <td className={styles.label}>
                <Label text={nodeData.environment} color="green" />
              </td>
              <td className={styles.inputs}>
                {nodeData.inputs.map((input, index) => (
                  <div key={`input-${index}`}>{input}</div>
                ))}
              </td>
              <td className={styles.outputs}>
                {nodeData.outputs.map((output, index) => (
                  <div key={`output-${index}`}>{output}</div>
                ))}
              </td>
              <td className={styles.version}>{nodeData.version}</td>
              <td className={styles.action}>
                <ActionButton text="View" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default NodesTable;
