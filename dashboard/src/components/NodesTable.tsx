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

  return (
    <div className={styles.tableContainer}>
      <table>
        <thead>
          <tr>
            {settings.showNodeId && <th>Node ID</th>}
            {settings.showName && <th>Name</th>}
            {settings.showType && <th>Type</th>}
            {settings.showEnv && <th>Environment</th>}
            {settings.showInputs && <th>Inputs From</th>}
            {settings.showOutputs && <th>Outputs</th>}
            {settings.showVersion && <th>Version</th>}
            {settings.showAction && <th>Action</th>}
          </tr>
        </thead>
        <tbody>
          {parsedData.map((nodeData, index) => (
            <tr key={index}>
              {settings.showNodeId && (
                <td className={styles.id}>{nodeData.id}</td>
              )}
              {settings.showName && (
                <td className={styles.name}>{nodeData.name}</td>
              )}
              {settings.showType && (
                <td className={styles.type}>{nodeData.type}</td>
              )}
              {settings.showEnv && (
                <td className={styles.label}>
                  {nodeData.environment && ( nodeData.environment)}
                </td>
              )}
              {settings.showInputs && (
                <td className={styles.inputs}>
                  {nodeData.inputs.map((input, index) => (
                    <div key={`input-${index}`}>{input}</div>
                  ))}
                </td>
              )}
              {settings.showOutputs && (
                <td className={styles.outputs}>
                  {nodeData.outputs.map((output, index) => (
                    <div key={`output-${index}`}>{output}</div>
                  ))}
                </td>
              )}
              {settings.showVersion && (
                <td className={styles.version}>{nodeData.version}</td>
              )}
              {settings.showAction && (
                <td className={styles.action}>
                  <ActionButton text="View" />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default NodesTable;
